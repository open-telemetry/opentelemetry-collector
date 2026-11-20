// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package reclaim // import "go.opentelemetry.io/collector/service/internal/reclaim"

import (
	"bufio"
	"debug/elf"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"math/bits"
	"os"
	"reflect"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/sys/unix"
)

const (
	auxvNull  = 0
	auxvEntry = 9 // AT_ENTRY from Linux include/uapi/linux/auxvec.h
)

type mapsEntry struct {
	start  uintptr
	end    uintptr
	perms  string
	offset uint64
	dev    string
	inode  uint64
	path   string
}

type targetRange struct {
	vaStart    uintptr
	vaEnd      uintptr
	fileOffset uint64
}

func (r targetRange) length() uint64 {
	if r.vaEnd <= r.vaStart {
		return 0
	}
	return uint64(r.vaEnd - r.vaStart)
}

type reclaimEnv struct {
	exePath         string
	mapsPath        string
	auxvPath        string
	pageSize        int
	anchorAddr      uintptr
	fstatfsType     func(fd int) (uint32, error)
	gcAndFree       func()
	madviseDontNeed func(vaStart uintptr, length uint64) error
	fadviseDontNeed func(fd int, offset, length int64) error
}

// ColdBinaryPages evicts cold startup binary pages from the Linux page cache
// after Collector initialization finishes.
//
// During startup, Go init() functions and component factory registration fault
// large portions of the Collector binary into Linux active_file cache, which
// does not age out without memory pressure and inflates container working set.
//
// This runs runtime.GC() and debug.FreeOSMemory() first (so GC stack scanning
// does not immediately refault .gopclntab/.text back in), then unmaps clean
// non-writable ELF PT_LOAD mappings with MADV_DONTNEED and drops their backing
// file ranges with byte-range-scoped posix_fadvise(POSIX_FADV_DONTNEED).
//
// Unmapping PTEs before posix_fadvise works unprivileged on root-owned
// container images (where MADV_PAGEOUT silently no-ops via can_do_file_pageout)
// and cooperates safely across pods sharing a container image layer (pages
// still mapped by sibling processes are left resident by the kernel).
func ColdBinaryPages() (Result, error) {
	return reclaimColdBinaryPages(reclaimEnv{
		exePath:         "/proc/self/exe",
		mapsPath:        "/proc/self/maps",
		auxvPath:        "/proc/self/auxv",
		pageSize:        os.Getpagesize(),
		anchorAddr:      reflect.ValueOf(anchorFunc).Pointer(),
		fstatfsType:     defaultFstatfsType,
		gcAndFree:       defaultGCAndFree,
		madviseDontNeed: defaultMadviseDontNeed,
		fadviseDontNeed: defaultFadviseDontNeed,
	})
}

func defaultFstatfsType(fd int) (uint32, error) {
	var st unix.Statfs_t
	if err := unix.Fstatfs(fd, &st); err != nil {
		return 0, err
	}
	return uint32(st.Type), nil
}

func defaultGCAndFree() {
	runtime.GC()
	debug.FreeOSMemory()
}

func defaultMadviseDontNeed(vaStart uintptr, length uint64) error {
	_, _, errno := unix.Syscall(unix.SYS_MADVISE, vaStart, uintptr(length), unix.MADV_DONTNEED)
	if errno != 0 {
		return fmt.Errorf("madvise(0x%x, %d, MADV_DONTNEED): %w", vaStart, length, errno)
	}
	return nil
}

func defaultFadviseDontNeed(fd int, offset, length int64) error {
	if err := unix.Fadvise(fd, offset, length, unix.FADV_DONTNEED); err != nil {
		return fmt.Errorf("fadvise(%d, %d, %d, FADV_DONTNEED): %w", fd, offset, length, err)
	}
	return nil
}

func isInMemoryFS(fsType uint32) bool {
	return fsType == uint32(unix.TMPFS_MAGIC) || fsType == uint32(unix.RAMFS_MAGIC)
}

func reclaimColdBinaryPages(env reclaimEnv) (Result, error) {
	if env.pageSize <= 0 || env.pageSize&(env.pageSize-1) != 0 {
		return Result{}, fmt.Errorf("invalid page size %d", env.pageSize)
	}
	pageSize := uint64(env.pageSize)

	f, err := os.Open(env.exePath)
	if err != nil {
		return Result{}, fmt.Errorf("open %s: %w", env.exePath, err)
	}
	defer f.Close()

	fd := int(f.Fd())
	if fsType, fsErr := env.fstatfsType(fd); fsErr == nil && isInMemoryFS(fsType) {
		return Result{SkippedReason: "in-memory filesystem"}, nil
	}

	ef, err := elf.NewFile(f)
	if err != nil {
		return Result{}, fmt.Errorf("parse ELF %s: %w", env.exePath, err)
	}
	hasTextRel, err := hasTextRelocations(ef)
	if err != nil {
		return Result{}, fmt.Errorf("check dynamic text relocations %s: %w", env.exePath, err)
	}
	if hasTextRel {
		return Result{SkippedReason: "text relocations present"}, nil
	}

	headers := make([]elf.ProgHeader, 0, len(ef.Progs))
	for _, p := range ef.Progs {
		headers = append(headers, p.ProgHeader)
	}
	cleanSafe := cleanELFIntervals(headers, pageSize)
	if len(cleanSafe) == 0 {
		return Result{SkippedReason: "no reclaimable ELF segments"}, nil
	}

	mapsBytes, err := os.ReadFile(env.mapsPath)
	if err != nil {
		return Result{}, fmt.Errorf("read %s: %w", env.mapsPath, err)
	}
	entries, err := parseProcMaps(string(mapsBytes))
	if err != nil {
		return Result{}, fmt.Errorf("parse %s: %w", env.mapsPath, err)
	}

	var statDev string
	var statInode uint64
	var st unix.Stat_t
	if statErr := unix.Fstat(fd, &st); statErr == nil {
		statDev = fmt.Sprintf("%02x:%02x", unix.Major(st.Dev), unix.Minor(st.Dev))
		statInode = st.Ino
	}
	resolvedPath, rlErr := os.Readlink(env.exePath)
	if rlErr != nil && errors.Is(rlErr, unix.EINVAL) {
		resolvedPath = env.exePath
	}

	var atEntry uintptr
	if env.auxvPath != "" {
		if auxvBytes, auxErr := os.ReadFile(env.auxvPath); auxErr == nil {
			atEntry, _ = parseAuxvEntry(auxvBytes, bits.UintSize/8, binary.NativeEndian)
		}
	}

	vmas, mismatch := filterExeVMAs(entries, env.anchorAddr, atEntry, statDev, statInode, resolvedPath)
	if mismatch {
		return Result{SkippedReason: "executable mapping mismatch"}, nil
	}
	targets := selectTargetRanges(vmas, cleanSafe, pageSize)
	if len(targets) == 0 {
		return Result{SkippedReason: "no reclaimable binary mappings"}, nil
	}

	// Run GC and release Go heap spans before evicting binary file pages so GC
	// stack scanning does not immediately refault .gopclntab and .text back in.
	env.gcAndFree()

	var res Result
	for _, t := range targets {
		length := t.length()
		if length == 0 || t.fileOffset > math.MaxInt64 || length > math.MaxInt64 {
			continue
		}
		if err := env.madviseDontNeed(t.vaStart, length); err != nil {
			return res, err
		}
		if err := env.fadviseDontNeed(fd, int64(t.fileOffset), int64(length)); err != nil {
			return res, err
		}
		res.RangesAdvised++
		res.BytesAdvised += length
	}
	return res, nil
}

// hasTextRelocations inspects PT_DYNAMIC program headers (working on stripped
// binaries without section headers) for DT_TEXTREL or DT_FLAGS & DF_TEXTREL.
// Binaries with text relocations have dirty COW pages inside r-xp/r--p PT_LOAD
// segments that MADV_DONTNEED would revert to unrelocated bytes.
func hasTextRelocations(ef *elf.File) (bool, error) {
	for _, p := range ef.Progs {
		if p.Type != elf.PT_DYNAMIC {
			continue
		}
		has, err := dynamicHasTextRel(p.Open(), ef.Class, ef.ByteOrder)
		if err != nil {
			return false, err
		}
		if has {
			return true, nil
		}
	}
	return false, nil
}

func dynamicHasTextRel(r io.Reader, class elf.Class, order binary.ByteOrder) (bool, error) {
	var entrySize int
	switch class {
	case elf.ELFCLASS32:
		entrySize = 8
	case elf.ELFCLASS64:
		entrySize = 16
	default:
		return false, fmt.Errorf("unsupported ELF class %v", class)
	}

	var buf [16]byte
	for {
		_, err := io.ReadFull(r, buf[:entrySize])
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("read PT_DYNAMIC: %w", err)
		}
		var tag, val uint64
		if class == elf.ELFCLASS32 {
			tag = uint64(order.Uint32(buf[0:4]))
			val = uint64(order.Uint32(buf[4:8]))
		} else {
			tag = order.Uint64(buf[0:8])
			val = order.Uint64(buf[8:16])
		}
		switch elf.DynTag(tag) {
		case elf.DT_NULL:
			return false, nil
		case elf.DT_TEXTREL:
			return true, nil
		case elf.DT_FLAGS:
			if (elf.DynFlag(val) & elf.DF_TEXTREL) != 0 {
				return true, nil
			}
		}
	}
}

// cleanELFIntervals derives page-aligned file-offset intervals for clean
// non-writable PT_LOAD segments, strictly excluding outward page-aligned
// PT_GNU_RELRO and writable PT_LOAD segments.
//
// Checking ELF program headers is required for both ET_DYN (PIE) and ET_EXEC
// (non-PIE) binaries because PT_GNU_RELRO regions are written during startup
// relocation and then mprotect'd to r--p; calling MADV_DONTNEED on those dirty
// COW pages reverts relocated pointers and crashes the process.
func cleanELFIntervals(progs []elf.ProgHeader, pageSize uint64) []interval {
	var cleanPTLoad []interval
	var excluded []interval

	for _, p := range progs {
		switch p.Type {
		case elf.PT_LOAD:
			if p.Filesz == 0 || p.Filesz > math.MaxUint64-p.Off {
				continue
			}
			if (p.Flags & elf.PF_W) != 0 {
				// Use p.Filesz (not p.Memsz) because .bss occupies no file
				// offsets; using Memsz could over-subtract clean segments at
				// higher file offsets.
				end, ok := alignUp(p.Off+p.Filesz, pageSize)
				if !ok {
					end = math.MaxUint64
				}
				excluded = append(excluded, interval{
					start: alignDown(p.Off, pageSize),
					end:   end,
				})
				continue
			}
			// Inward-align clean PT_LOAD ranges so every advised page is
			// strictly backed by clean read-only file bytes.
			start, ok := alignUp(p.Off, pageSize)
			if !ok {
				continue
			}
			end := alignDown(p.Off+p.Filesz, pageSize)
			if end > start {
				cleanPTLoad = append(cleanPTLoad, interval{start: start, end: end})
			}
		case elf.PT_GNU_RELRO:
			span := max(p.Memsz, p.Filesz)
			if span == 0 || span > math.MaxUint64-p.Off {
				continue
			}
			end, ok := alignUp(p.Off+span, pageSize)
			if !ok {
				end = math.MaxUint64
			}
			excluded = append(excluded, interval{
				start: alignDown(p.Off, pageSize),
				end:   end,
			})
		}
	}

	return subtractIntervals(cleanPTLoad, excluded)
}

// anchorFunc provides an in-binary code PC so filterExeVMAs can identify the
// executable's dev:inode directly from /proc/self/maps, avoiding overlayfs
// dev:inode mismatches against stat("/proc/self/exe") on kernels < 6.7.
func anchorFunc() {}

func cleanPath(p string) string {
	return strings.TrimSuffix(strings.TrimSpace(p), " (deleted)")
}

func parseAuxvEntry(data []byte, ptrSize int, order binary.ByteOrder) (uintptr, bool) {
	if ptrSize != 4 && ptrSize != 8 {
		return 0, false
	}
	step := ptrSize * 2
	maxPtr := uint64(^uintptr(0))
	for i := 0; i+step <= len(data); i += step {
		var tag, val uint64
		if ptrSize == 4 {
			tag = uint64(order.Uint32(data[i : i+4]))
			val = uint64(order.Uint32(data[i+4 : i+8]))
		} else {
			tag = order.Uint64(data[i : i+8])
			val = order.Uint64(data[i+8 : i+16])
		}
		if tag == auxvNull {
			break
		}
		if tag == auxvEntry && val != 0 && val <= maxPtr {
			return uintptr(val), true
		}
	}
	return 0, false
}

func parseProcMaps(content string) ([]mapsEntry, error) {
	maxPtr := uint64(^uintptr(0))
	var out []mapsEntry
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		startStr, endStr, ok := strings.Cut(fields[0], "-")
		if !ok {
			continue
		}
		start, err1 := strconv.ParseUint(startStr, 16, 64)
		end, err2 := strconv.ParseUint(endStr, 16, 64)
		if err1 != nil || err2 != nil || end <= start || end > maxPtr {
			continue
		}
		offset, err := strconv.ParseUint(fields[2], 16, 64)
		if err != nil {
			continue
		}
		inode, err := strconv.ParseUint(fields[4], 10, 64)
		if err != nil {
			continue
		}
		path := ""
		if len(fields) >= 6 {
			rest := line
			for i := range 5 {
				rest = strings.TrimLeftFunc(rest, unicode.IsSpace)
				rest = rest[len(fields[i]):]
			}
			path = strings.TrimSpace(rest)
		}
		out = append(out, mapsEntry{
			start:  uintptr(start),
			end:    uintptr(end),
			perms:  fields[1],
			offset: offset,
			dev:    fields[3],
			inode:  inode,
			path:   path,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func findVMAForAddr(entries []mapsEntry, addr uintptr) (mapsEntry, bool) {
	if addr == 0 {
		return mapsEntry{}, false
	}
	for _, e := range entries {
		if addr >= e.start && addr < e.end && e.inode != 0 {
			return e, true
		}
	}
	return mapsEntry{}, false
}

// filterExeVMAs returns the /proc/self/maps VMAs belonging to /proc/self/exe.
//
// When anchorAddr is non-zero, it verifies that anchorFunc resides in the same
// backing file as /proc/self/exe (using AT_ENTRY from /proc/self/auxv when
// available, or falling back to fstat dev:inode / resolved path). This prevents
// intersecting /proc/self/exe's ELF headers with a different binary's VMAs when
// launched via an explicit dynamic loader (ld.so) or embedded in a shared
// library (-buildmode=c-shared / plugin).
func filterExeVMAs(entries []mapsEntry, anchorAddr, atEntry uintptr, statDev string, statInode uint64, resolvedExePath string) (vmas []mapsEntry, mismatch bool) {
	var targetDev string
	var targetInode uint64
	var found bool

	if anchorAddr != 0 {
		anchorVMA, ok := findVMAForAddr(entries, anchorAddr)
		if !ok {
			return nil, true
		}
		if atEntry != 0 {
			entryVMA, entryOK := findVMAForAddr(entries, atEntry)
			if !entryOK || entryVMA.dev != anchorVMA.dev || entryVMA.inode != anchorVMA.inode {
				return nil, true
			}
		} else {
			devMatch := statInode != 0 && anchorVMA.dev == statDev && anchorVMA.inode == statInode
			pathMatch := resolvedExePath != "" && anchorVMA.path != "" && !strings.HasPrefix(anchorVMA.path, "[") && cleanPath(anchorVMA.path) == cleanPath(resolvedExePath)
			if !devMatch && !pathMatch {
				return nil, true
			}
		}
		targetDev = anchorVMA.dev
		targetInode = anchorVMA.inode
		found = true
	} else if entryVMA, ok := findVMAForAddr(entries, atEntry); ok {
		targetDev = entryVMA.dev
		targetInode = entryVMA.inode
		found = true
	} else if statInode != 0 {
		targetDev = statDev
		targetInode = statInode
		found = true
	}

	var matched []mapsEntry
	if found {
		for _, e := range entries {
			if e.inode == targetInode && e.dev == targetDev {
				matched = append(matched, e)
			}
		}
	}
	if len(matched) == 0 && resolvedExePath != "" {
		cleanTarget := cleanPath(resolvedExePath)
		for _, e := range entries {
			if cleanPath(e.path) == cleanTarget && e.path != "" && !strings.HasPrefix(e.path, "[") {
				matched = append(matched, e)
			}
		}
	}
	return matched, false
}

func selectTargetRanges(vmas []mapsEntry, cleanSafe []interval, pageSize uint64) []targetRange {
	var out []targetRange
	for _, v := range vmas {
		// Only readable, non-writable private mappings (r-xp or r--p).
		if len(v.perms) < 4 || v.perms[0] != 'r' || v.perms[1] != '-' || v.perms[3] != 'p' {
			continue
		}
		vmaLen := uint64(v.end - v.start)
		if vmaLen > math.MaxUint64-v.offset {
			continue
		}
		vmaFile := interval{start: v.offset, end: v.offset + vmaLen}

		for _, safe := range cleanSafe {
			ov, ok := intersectInterval(vmaFile, safe)
			if !ok {
				continue
			}
			startOff, ok := alignUp(ov.start, pageSize)
			if !ok {
				continue
			}
			endOff := alignDown(ov.end, pageSize)
			if endOff <= startOff {
				continue
			}
			vaStart := v.start + uintptr(startOff-v.offset)
			vaEnd := v.start + uintptr(endOff-v.offset)
			if vaStart < v.start || vaEnd > v.end || vaEnd <= vaStart {
				continue
			}
			out = append(out, targetRange{
				vaStart:    vaStart,
				vaEnd:      vaEnd,
				fileOffset: startOff,
			})
		}
	}
	return out
}

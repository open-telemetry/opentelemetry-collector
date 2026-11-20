// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package reclaim

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"errors"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestCleanELFIntervals(t *testing.T) {
	t.Parallel()

	const ps = 4096
	progs := []elf.ProgHeader{
		// Unaligned clean start and end: [0x1400, 0x10400) -> inward [0x2000, 0x10000).
		{Type: elf.PT_LOAD, Flags: elf.PF_R | elf.PF_X, Off: 0x1400, Filesz: 0xf000, Memsz: 0xf000},
		// Sub-page clean segment [0x200, 0x800): inward-aligns to empty and is skipped.
		{Type: elf.PT_LOAD, Flags: elf.PF_R, Off: 0x200, Filesz: 0x600, Memsz: 0x600},
		// Clean read-only segment [0x10000, 0x30000) containing an unaligned PT_GNU_RELRO
		// where Memsz > Filesz, and sharing boundary page [0x2f000, 0x30000) with writable PT_LOAD.
		{Type: elf.PT_LOAD, Flags: elf.PF_R, Off: 0x10000, Filesz: 0x20000, Memsz: 0x20000},
		// Unaligned PT_GNU_RELRO [0x15960, 0x16800) with Memsz=0x1aa0 -> end 0x17400 -> outward [0x15000, 0x18000).
		{Type: elf.PT_GNU_RELRO, Flags: elf.PF_R, Off: 0x15960, Filesz: 0xea0, Memsz: 0x1aa0},
		// Writable data segment starting mid-page at 0x2f800 (excludes shared page [0x2f000, 0x30000))
		// with Filesz=0x2a00 (outward [0x2f000, 0x33000)) and huge .bss Memsz=0x50000.
		{Type: elf.PT_LOAD, Flags: elf.PF_R | elf.PF_W, Off: 0x2f800, Filesz: 0x2a00, Memsz: 0x50000},
		// Higher clean segment at 0x40000 must NOT be subtracted by writable .bss Memsz.
		{Type: elf.PT_LOAD, Flags: elf.PF_R, Off: 0x40000, Filesz: 0x5000, Memsz: 0x5000},
		// Malformed overflow header safely ignored.
		{Type: elf.PT_LOAD, Flags: elf.PF_R, Off: math.MaxUint64 - 16, Filesz: 64, Memsz: 64},
	}

	got := cleanELFIntervals(progs, ps)
	assert.Equal(t, []interval{
		{start: 0x2000, end: 0x15000},
		{start: 0x18000, end: 0x2f000},
		{start: 0x40000, end: 0x45000},
	}, got)
}

func TestDynamicHasTextRel(t *testing.T) {
	t.Parallel()

	encode64 := func(order binary.ByteOrder, pairs ...[2]uint64) []byte {
		var buf bytes.Buffer
		for _, p := range pairs {
			_ = binary.Write(&buf, order, p[0])
			_ = binary.Write(&buf, order, p[1])
		}
		return buf.Bytes()
	}
	encode32 := func(order binary.ByteOrder, pairs ...[2]uint32) []byte {
		var buf bytes.Buffer
		for _, p := range pairs {
			_ = binary.Write(&buf, order, p[0])
			_ = binary.Write(&buf, order, p[1])
		}
		return buf.Bytes()
	}

	t.Run("64bit_dt_textrel", func(t *testing.T) {
		t.Parallel()
		raw := encode64(binary.LittleEndian, [2]uint64{uint64(elf.DT_TEXTREL), 0}, [2]uint64{uint64(elf.DT_NULL), 0})
		got, err := dynamicHasTextRel(bytes.NewReader(raw), elf.ELFCLASS64, binary.LittleEndian)
		require.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("64bit_df_textrel_big_endian", func(t *testing.T) {
		t.Parallel()
		raw := encode64(binary.BigEndian, [2]uint64{uint64(elf.DT_FLAGS), uint64(elf.DF_TEXTREL)}, [2]uint64{uint64(elf.DT_NULL), 0})
		got, err := dynamicHasTextRel(bytes.NewReader(raw), elf.ELFCLASS64, binary.BigEndian)
		require.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("32bit_clean_stops_at_dt_null", func(t *testing.T) {
		t.Parallel()
		raw := encode32(binary.LittleEndian,
			[2]uint32{uint32(elf.DT_FLAGS), uint32(elf.DF_BIND_NOW)},
			[2]uint32{uint32(elf.DT_NULL), 0},
			[2]uint32{uint32(elf.DT_TEXTREL), 0}, // after DT_NULL: ignored
		)
		got, err := dynamicHasTextRel(bytes.NewReader(raw), elf.ELFCLASS32, binary.LittleEndian)
		require.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("truncated_returns_error", func(t *testing.T) {
		t.Parallel()
		_, err := dynamicHasTextRel(bytes.NewReader([]byte{1, 2, 3}), elf.ELFCLASS64, binary.LittleEndian)
		require.Error(t, err)
	})
}

func TestParseAuxvEntry(t *testing.T) {
	t.Parallel()

	var buf64 bytes.Buffer
	_ = binary.Write(&buf64, binary.LittleEndian, uint64(6))
	_ = binary.Write(&buf64, binary.LittleEndian, uint64(4096))
	_ = binary.Write(&buf64, binary.LittleEndian, uint64(auxvEntry))
	_ = binary.Write(&buf64, binary.LittleEndian, uint64(0x401234))
	_ = binary.Write(&buf64, binary.LittleEndian, uint64(auxvNull))
	_ = binary.Write(&buf64, binary.LittleEndian, uint64(0))

	entry, ok := parseAuxvEntry(buf64.Bytes(), 8, binary.LittleEndian)
	assert.True(t, ok)
	assert.Equal(t, uintptr(0x401234), entry)

	var buf32 bytes.Buffer
	_ = binary.Write(&buf32, binary.BigEndian, uint32(auxvEntry))
	_ = binary.Write(&buf32, binary.BigEndian, uint32(0x8048100))
	_ = binary.Write(&buf32, binary.BigEndian, uint32(auxvNull))
	_ = binary.Write(&buf32, binary.BigEndian, uint32(0))

	entry, ok = parseAuxvEntry(buf32.Bytes(), 4, binary.BigEndian)
	assert.True(t, ok)
	assert.Equal(t, uintptr(0x8048100), entry)

	_, ok = parseAuxvEntry(nil, 8, binary.LittleEndian)
	assert.False(t, ok)
}

func TestParseProcMaps(t *testing.T) {
	t.Parallel()

	sample := `
00400000-00500000 r-xp 00001000 fc:01 12345 /opt/my collector/otelcol (deleted)
00500000-00600000 r--p 00101000 fc:01 12345 /opt/my collector/otelcol (deleted)
00600000-00610000 rw-p 00201000 fc:01 12345 /opt/my collector/otelcol (deleted)
00700000-00710000 rw-p 00000000 00:00 0 [heap]
malformed line
`
	entries, err := parseProcMaps(sample)
	require.NoError(t, err)
	require.Len(t, entries, 4)

	assert.Equal(t, uintptr(0x400000), entries[0].start)
	assert.Equal(t, uintptr(0x500000), entries[0].end)
	assert.Equal(t, "r-xp", entries[0].perms)
	assert.Equal(t, uint64(0x1000), entries[0].offset)
	assert.Equal(t, "fc:01", entries[0].dev)
	assert.Equal(t, uint64(12345), entries[0].inode)
	assert.Equal(t, "/opt/my collector/otelcol (deleted)", entries[0].path)
	assert.Equal(t, "/opt/my collector/otelcol", cleanPath(entries[0].path))
	assert.Equal(t, "[heap]", entries[3].path)
}

func TestFilterExeVMAs(t *testing.T) {
	t.Parallel()

	entries := []mapsEntry{
		{start: 0x400000, end: 0x500000, perms: "r-xp", offset: 0x1000, dev: "00:42", inode: 999, path: "/otelcol (deleted)"},
		{start: 0x500000, end: 0x600000, perms: "r--p", offset: 0x101000, dev: "00:42", inode: 999, path: "/otelcol (deleted)"},
		{start: 0x700000, end: 0x800000, perms: "r-xp", offset: 0x0, dev: "fc:01", inode: 777, path: "/lib64/ld-linux-x86-64.so.2"},
	}

	// Anchor PC and AT_ENTRY agree on dev:inode even when fstat(dev, inode) disagrees (overlayfs < 6.7).
	got, mismatch := filterExeVMAs(entries, 0x450000, 0x401000, "fc:02", 888, "/wrong")
	assert.False(t, mismatch)
	require.Len(t, got, 2)
	assert.Equal(t, uintptr(0x400000), got[0].start)
	assert.Equal(t, uintptr(0x500000), got[1].start)

	// Anchor PC in /otelcol vs AT_ENTRY in ld-linux (explicit loader or c-shared) flags mismatch.
	got, mismatch = filterExeVMAs(entries, 0x450000, 0x710000, "fc:01", 777, "/lib64/ld-linux-x86-64.so.2")
	assert.True(t, mismatch)
	assert.Empty(t, got)

	// When auxv is unreadable (atEntry == 0), anchor PC still requires dev:inode or path match.
	got, mismatch = filterExeVMAs(entries, 0x450000, 0, "00:42", 999, "/wrong")
	assert.False(t, mismatch)
	require.Len(t, got, 2)

	got, mismatch = filterExeVMAs(entries, 0x450000, 0, "fc:99", 111, "/otelcol")
	assert.False(t, mismatch)
	require.Len(t, got, 2)

	got, mismatch = filterExeVMAs(entries, 0x450000, 0, "fc:01", 777, "/lib64/ld-linux-x86-64.so.2")
	assert.True(t, mismatch)
	assert.Empty(t, got)

	// Fallback to AT_ENTRY / stat(dev, inode) / cleaned path when anchorAddr == 0.
	got, mismatch = filterExeVMAs(entries, 0, 0, "00:42", 999, "/wrong")
	assert.False(t, mismatch)
	require.Len(t, got, 2)

	got, mismatch = filterExeVMAs(entries, 0, 0, "fc:99", 111, "/otelcol")
	assert.False(t, mismatch)
	require.Len(t, got, 2)
}

func TestSelectTargetRanges(t *testing.T) {
	t.Parallel()

	const ps = 4096
	cleanSafe := []interval{
		{start: 0x1000, end: 0x20000},
		{start: 0x25000, end: 0x30000}, // [0x20000, 0x25000) excluded as PT_GNU_RELRO
	}
	vmas := []mapsEntry{
		{start: 0x401000, end: 0x410000, perms: "r-xp", offset: 0x1000},  // clean code
		{start: 0x410000, end: 0x430000, perms: "r--p", offset: 0x10000}, // split around RELRO hole
		{start: 0x430000, end: 0x440000, perms: "rw-p", offset: 0x25000}, // writable: must skip
		{start: 0x440000, end: 0x450000, perms: "---p", offset: 0x25000}, // guard: must skip
		{start: 0x450000, end: 0x460000, perms: "r--s", offset: 0x25000}, // shared: must skip
	}

	got := selectTargetRanges(vmas, cleanSafe, ps)
	assert.Equal(t, []targetRange{
		{vaStart: 0x401000, vaEnd: 0x410000, fileOffset: 0x1000},
		{vaStart: 0x410000, vaEnd: 0x420000, fileOffset: 0x10000},
		{vaStart: 0x425000, vaEnd: 0x430000, fileOffset: 0x25000},
	}, got)
}

func TestIsInMemoryFS(t *testing.T) {
	t.Parallel()

	assert.True(t, isInMemoryFS(uint32(unix.TMPFS_MAGIC)))
	assert.True(t, isInMemoryFS(uint32(unix.RAMFS_MAGIC)))
	assert.False(t, isInMemoryFS(uint32(unix.EXT4_SUPER_MAGIC)))
	assert.False(t, isInMemoryFS(uint32(unix.OVERLAYFS_SUPER_MAGIC)))
}

func TestReclaimColdBinaryPagesMocked(t *testing.T) {
	t.Parallel()

	const (
		exePath  = "/proc/self/exe"
		mapsPath = "/proc/self/maps"
		auxvPath = "/proc/self/auxv"
	)
	anchorAddr := reflect.ValueOf(anchorFunc).Pointer()
	pageSize := os.Getpagesize()

	t.Run("orders_gc_before_madvise_and_fadvise_with_valid_ranges", func(t *testing.T) {
		t.Parallel()

		type fadviseCall struct {
			offset int64
			length int64
		}
		var events []string
		var madvLengths []uint64
		var fadvCalls []fadviseCall

		env := reclaimEnv{
			exePath:     exePath,
			mapsPath:    mapsPath,
			auxvPath:    auxvPath,
			pageSize:    pageSize,
			anchorAddr:  anchorAddr,
			fstatfsType: func(int) (uint32, error) { return uint32(unix.EXT4_SUPER_MAGIC), nil },
			gcAndFree:   func() { events = append(events, "gc") },
			madviseDontNeed: func(vaStart uintptr, length uint64) error {
				assert.Zero(t, uint64(vaStart)%uint64(pageSize))
				assert.Zero(t, length%uint64(pageSize))
				assert.Positive(t, length)
				events = append(events, "madvise")
				madvLengths = append(madvLengths, length)
				return nil
			},
			fadviseDontNeed: func(_ int, offset, length int64) error {
				assert.GreaterOrEqual(t, offset, int64(0))
				assert.Zero(t, offset%int64(pageSize))
				assert.Positive(t, length)
				assert.Zero(t, length%int64(pageSize))
				events = append(events, "fadvise")
				fadvCalls = append(fadvCalls, fadviseCall{offset: offset, length: length})
				return nil
			},
		}

		res, err := reclaimColdBinaryPages(env)
		require.NoError(t, err)
		assert.Empty(t, res.SkippedReason)
		assert.Positive(t, res.RangesAdvised)
		require.Len(t, madvLengths, res.RangesAdvised)
		require.Len(t, fadvCalls, res.RangesAdvised)

		expectedEvents := make([]string, 0, 1+2*res.RangesAdvised)
		expectedEvents = append(expectedEvents, "gc")
		var total uint64
		for i := range res.RangesAdvised {
			expectedEvents = append(expectedEvents, "madvise", "fadvise")
			assert.Equal(t, int64(madvLengths[i]), fadvCalls[i].length)
			total += madvLengths[i]
		}
		assert.Equal(t, expectedEvents, events)
		assert.Equal(t, total, res.BytesAdvised)
	})

	t.Run("skips_in_memory_fs_without_running_gc", func(t *testing.T) {
		t.Parallel()

		gcCalled := false
		env := reclaimEnv{
			exePath:     exePath,
			mapsPath:    mapsPath,
			auxvPath:    auxvPath,
			pageSize:    pageSize,
			anchorAddr:  anchorAddr,
			fstatfsType: func(int) (uint32, error) { return uint32(unix.TMPFS_MAGIC), nil },
			gcAndFree:   func() { gcCalled = true },
		}

		res, err := reclaimColdBinaryPages(env)
		require.NoError(t, err)
		assert.Equal(t, "in-memory filesystem", res.SkippedReason)
		assert.False(t, gcCalled)
	})

	t.Run("skips_on_executable_mapping_mismatch", func(t *testing.T) {
		t.Parallel()

		// Anchor points to an unmapped / mismatched address while auxv points to real /proc/self/exe.
		gcCalled := false
		env := reclaimEnv{
			exePath:     exePath,
			mapsPath:    mapsPath,
			auxvPath:    auxvPath,
			pageSize:    pageSize,
			anchorAddr:  0x1,
			fstatfsType: func(int) (uint32, error) { return uint32(unix.EXT4_SUPER_MAGIC), nil },
			gcAndFree:   func() { gcCalled = true },
		}

		res, err := reclaimColdBinaryPages(env)
		require.NoError(t, err)
		assert.Equal(t, "executable mapping mismatch", res.SkippedReason)
		assert.False(t, gcCalled)
	})

	t.Run("skips_when_no_matching_vmas", func(t *testing.T) {
		t.Parallel()

		emptyMaps := filepath.Join(t.TempDir(), "empty_maps")
		require.NoError(t, os.WriteFile(emptyMaps, []byte("00700000-00710000 rw-p 00000000 00:00 0 [heap]\n"), 0o600))

		gcCalled := false
		env := reclaimEnv{
			exePath:     exePath,
			mapsPath:    emptyMaps,
			auxvPath:    "",
			pageSize:    pageSize,
			anchorAddr:  0,
			fstatfsType: func(int) (uint32, error) { return uint32(unix.EXT4_SUPER_MAGIC), nil },
			gcAndFree:   func() { gcCalled = true },
		}

		res, err := reclaimColdBinaryPages(env)
		require.NoError(t, err)
		assert.Equal(t, "no reclaimable binary mappings", res.SkippedReason)
		assert.False(t, gcCalled)
	})

	t.Run("propagates_madvise_and_fadvise_errors", func(t *testing.T) {
		t.Parallel()

		madvErr := errors.New("madvise boom")
		fadvCalled := false
		env := reclaimEnv{
			exePath:     exePath,
			mapsPath:    mapsPath,
			auxvPath:    auxvPath,
			pageSize:    pageSize,
			anchorAddr:  anchorAddr,
			fstatfsType: func(int) (uint32, error) { return uint32(unix.EXT4_SUPER_MAGIC), nil },
			gcAndFree:   func() {},
			madviseDontNeed: func(uintptr, uint64) error {
				return madvErr
			},
			fadviseDontNeed: func(int, int64, int64) error {
				fadvCalled = true
				return nil
			},
		}
		_, err := reclaimColdBinaryPages(env)
		require.ErrorIs(t, err, madvErr)
		assert.False(t, fadvCalled)

		fadvErr := errors.New("fadvise boom")
		env.madviseDontNeed = func(uintptr, uint64) error { return nil }
		env.fadviseDontNeed = func(int, int64, int64) error { return fadvErr }
		_, err = reclaimColdBinaryPages(env)
		require.ErrorIs(t, err, fadvErr)
	})

	t.Run("rejects_invalid_page_size_and_non_elf", func(t *testing.T) {
		t.Parallel()

		_, err := reclaimColdBinaryPages(reclaimEnv{pageSize: 3000})
		require.Error(t, err)

		nonELF := filepath.Join(t.TempDir(), "not_an_elf")
		require.NoError(t, os.WriteFile(nonELF, []byte("plain text"), 0o600))
		_, err = reclaimColdBinaryPages(reclaimEnv{
			exePath:     nonELF,
			mapsPath:    mapsPath,
			pageSize:    pageSize,
			fstatfsType: func(int) (uint32, error) { return uint32(unix.EXT4_SUPER_MAGIC), nil },
		})
		require.Error(t, err)
	})

	t.Run("returns_error_on_unreadable_exe", func(t *testing.T) {
		t.Parallel()

		if os.Geteuid() == 0 {
			t.Skip("root bypasses file read restrictions")
		}
		unreadable := filepath.Join(t.TempDir(), "unreadable_exe")
		require.NoError(t, os.WriteFile(unreadable, []byte("x"), 0o600))
		require.NoError(t, os.Chmod(unreadable, 0o000))
		t.Cleanup(func() {
			_ = os.Chmod(unreadable, 0o600)
		})

		_, err := reclaimColdBinaryPages(reclaimEnv{
			exePath:  unreadable,
			mapsPath: mapsPath,
			pageSize: pageSize,
		})
		require.ErrorIs(t, err, os.ErrPermission)
	})
}

func TestColdBinaryPagesLive(t *testing.T) {
	t.Parallel()

	res, err := ColdBinaryPages()
	require.NoError(t, err)
	if res.SkippedReason != "" {
		assert.Equal(t, "in-memory filesystem", res.SkippedReason)
		return
	}
	assert.Positive(t, res.RangesAdvised, "expected advised ranges on live binary, got %+v", res)
	assert.Positive(t, res.BytesAdvised)
}

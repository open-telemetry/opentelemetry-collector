package processscraper

import "sort"

func NewProcessInfo(name string, value float64) *processInfo {
	return &processInfo{name, value}
}

type processInfoList []*processInfo

// Len implements sort.Interface.
func (p processInfoList) Len() int {
	return len(p)
}

// Less implements sort.Interface.
func (p processInfoList) Less(i int, j int) bool {
	return p[i].Value > p[j].Value
}

// Swap implements sort.Interface.
func (p processInfoList) Swap(i int, j int) {
	p[i], p[j] = p[j], p[i]
}

var _ sort.Interface = (processInfoList)(nil)

func NewTopK(k uint) TopKInterface {
	if k == 0 {
		k = 10
	}
	return &topK{
		processes: make(processInfoList, 0),
		k:         int(k),
	}
}

type processInfo struct {
	Name  string
	Value float64
}

type TopKInterface interface {
	Append(p *processInfo)
	GetTop() map[string]struct{}
}

// 基于排序实现计算TopK值
type topK struct {
	// 永远有序，而且长度为K
	processes processInfoList
	k         int
}

func (t *topK) Append(p *processInfo) {
	t.processes = append(t.processes, p)
}

/// 返回值数量小于等于k，因为会有相同进程名的进程
func (t *topK) GetTop() map[string]struct{} {
	sort.Sort(t.processes)
	result := make(map[string]struct{})
	for i, process := range t.processes {
		if i == t.k {
			break
		}
		result[process.Name] = struct{}{}
	}
	return result
}

var _ TopKInterface = (*topK)(nil)

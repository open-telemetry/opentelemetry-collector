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
		K:         int(k),
	}
}

type processInfo struct {
	Name  string
	Value float64
}

type TopKInterface interface {
	Append(p *processInfo) bool
}

// 基于排序实现计算TopK值
type topK struct {
	// 永远有序，而且长度为K
	processes processInfoList
	K         int
}

func (t *topK) Append(p *processInfo) bool {
	if exists, i := t.in(p); exists {
		t.processes[i] = p
	} else {
		t.processes = append(t.processes, p)
	}
	t.calculate()
	exists, _ := t.in(p)
	return exists
}

func (t *topK) in(p *processInfo) (bool, int) {
	for i, process := range t.processes {
		if process.Name == p.Name {
			return true, i
		}
	}
	return false, -1
}

func (t *topK) calculate() {
	// 先排序
	sort.Sort(t.processes)
	// 数组长度小于k,直接返回
	if len(t.processes) < t.K {
		return
	}
	t.processes = t.processes[0:t.K]
}

var _ TopKInterface = (*topK)(nil)

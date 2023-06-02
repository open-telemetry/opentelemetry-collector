package processscraper

import (
	"reflect"
	"testing"
)

func TestNewProcessInfo(t *testing.T) {
	type args struct {
		name  string
		value float64
	}
	tests := []struct {
		name string
		args args
		want *processInfo
	}{
		{
			name: "ok",
			args: args{},
			want: &processInfo{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewProcessInfo(tt.args.name, tt.args.value); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewProcessInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewTopK(t *testing.T) {
	type args struct {
		k uint
	}
	tests := []struct {
		name string
		args args
		want *topK
	}{
		{
			name: "ok, k = 0",
			args: args{k: 0},
			want: &topK{
				processes: make([]*processInfo, 0),
				K:         10,
			},
		},
		{
			name: "ok, k = 123",
			args: args{k: 123},
			want: &topK{
				processes: make([]*processInfo, 0),
				K:         123,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewTopK(tt.args.k); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTopK() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_topK_Append(t *testing.T) {
	type args struct {
		p *processInfo
	}
	tests := []struct {
		name     string
		tr       *topK
		args     args
		want     bool
		wantData processInfoList
	}{
		{
			name: "success, append",
			tr: &topK{
				processes: processInfoList{
					{
						Name:  "foo3",
						Value: 3.0,
					},
					{
						Name:  "foo2",
						Value: 2.0,
					},
					{
						Name:  "foo1",
						Value: 1.0,
					},
				},
				K: 4,
			},
			args: args{
				&processInfo{
					Name:  "foo4",
					Value: 4.0,
				},
			},
			want: true,
			wantData: processInfoList{
				{
					Name:  "foo4",
					Value: 4.0,
				},
				{
					Name:  "foo3",
					Value: 3.0,
				},
				{
					Name:  "foo2",
					Value: 2.0,
				},
				{
					Name:  "foo1",
					Value: 1.0,
				},
			},
		},
		{
			name: "success, replace",
			tr: &topK{
				processes: processInfoList{
					{
						Name:  "foo3",
						Value: 3.0,
					},
					{
						Name:  "foo2",
						Value: 2.0,
					},
					{
						Name:  "foo1",
						Value: 1.0,
					},
				},
				K: 3,
			},
			args: args{
				&processInfo{
					Name:  "foo4",
					Value: 4.0,
				},
			},
			want: true,
			wantData: processInfoList{
				{
					Name:  "foo4",
					Value: 4.0,
				},
				{
					Name:  "foo3",
					Value: 3.0,
				},
				{
					Name:  "foo2",
					Value: 2.0,
				},
			},
		},
		{
			name: "success, resort",
			tr: &topK{
				processes: processInfoList{
					{
						Name:  "foo3",
						Value: 3.0,
					},
					{
						Name:  "foo2",
						Value: 2.0,
					},
					{
						Name:  "foo1",
						Value: 1.0,
					},
				},
				K: 3,
			},
			args: args{
				&processInfo{
					Name:  "foo3",
					Value: 1.5,
				},
			},
			want: true,
			wantData: processInfoList{
				{
					Name:  "foo2",
					Value: 2.0,
				},
				{
					Name:  "foo3",
					Value: 1.5,
				},
				{
					Name:  "foo1",
					Value: 1.0,
				},
			},
		},
		{
			name: "success, no effect",
			tr: &topK{
				processes: processInfoList{
					{
						Name:  "foo3",
						Value: 3.0,
					},
					{
						Name:  "foo2",
						Value: 2.0,
					},
					{
						Name:  "foo1",
						Value: 1.0,
					},
				},
				K: 3,
			},
			args: args{
				&processInfo{
					Name:  "foo4",
					Value: 0.01,
				},
			},
			want: false,
			wantData: processInfoList{
				{
					Name:  "foo3",
					Value: 3.0,
				},
				{
					Name:  "foo2",
					Value: 2.0,
				},
				{
					Name:  "foo1",
					Value: 1.0,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got1 := tt.tr.Append(tt.args.p); got1 != tt.want {
				t.Errorf("topK.Append() = %v, want %v", got1, tt.want)
			}
			if got2 := tt.tr.processes; !reflect.DeepEqual(got2, tt.wantData) {
				t.Errorf("topK.processes = %v, want %v", got2, tt.wantData)
			}
		})
	}
}

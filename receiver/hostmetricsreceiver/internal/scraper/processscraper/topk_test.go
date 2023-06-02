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
				k:         10,
			},
		},
		{
			name: "ok, k = 123",
			args: args{k: 123},
			want: &topK{
				processes: make([]*processInfo, 0),
				k:         123,
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
				k: 5,
			},
			args: args{
				&processInfo{
					Name:  "foo4",
					Value: 4.0,
				},
			},
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
				{
					Name:  "foo4",
					Value: 4.0,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.tr.Append(tt.args.p)
			if got2 := tt.tr.processes; !reflect.DeepEqual(got2, tt.wantData) {
				t.Errorf("topK.processes = %v, want %v", got2, tt.wantData)
			}
		})
	}
}

func Test_topK_GetTop(t *testing.T) {
	type fields struct {
		processes processInfoList
		k         int
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]struct{}
	}{
		{
			name: "ok",
			fields: fields{
				processes: processInfoList{
					{
						Name:  "foo1",
						Value: 1.0,
					},
					{
						Name:  "foo2",
						Value: 2.0,
					},
					{
						Name:  "foo3",
						Value: 3.0,
					},
					{
						Name:  "foo3",
						Value: 4.0,
					},
				},
				k: 2,
			},
			want: map[string]struct{}{"foo3": {}},
		},
		{
			name: "ok 2",
			fields: fields{
				processes: processInfoList{
					{
						Name:  "foo1",
						Value: 1.0,
					},
					{
						Name:  "foo2",
						Value: 2.0,
					},
					{
						Name:  "foo3",
						Value: 3.0,
					},
					{
						Name:  "foo3",
						Value: 0.1,
					},
					{
						Name:  "foo4",
						Value: 5.0,
					},
				},
				k: 3,
			},
			want: map[string]struct{}{"foo3": {}, "foo4": {}, "foo2": {}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &topK{
				processes: tt.fields.processes,
				k:         tt.fields.k,
			}
			if got := tr.GetTop(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("topK.GetTop() = %v, want %v", got, tt.want)
			}
		})
	}
}

package zutils

import "testing"

func Test_fnv64a_Sum64(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{
			name: "test1",
			args: args{
				key: "test1",
			},
			want: 2271358237066212092,
		},
		{
			name: "test2",
			args: args{
				key: "test2",
			},
			want: 2271361535601096725,
		},
	}

	f := fnv64a{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := f.Sum64(tt.args.key); got != tt.want {
				t.Errorf("fnv64a.Sum64() = %v, want %v", got, tt.want)
			}
		})
	}
}

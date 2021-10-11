package utils

import "testing"

func Test_isValidImportFile(t *testing.T) {
	type args struct {
		fileName string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"hidden file", args{".file"}, false},
		{"hidden folder", args{".folder/"}, false},
		{"non yaml/yml file", args{"hello.txt"}, false},
		{"non yaml/yml file", args{"foobar"}, false},
		{"valid file", args{"topic.yml"}, true},
		{"valid file", args{"topic.yaml"}, true},
		{"invalid dir entry", args{"folder/"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidImportFile(tt.args.fileName); got != tt.want {
				t.Errorf("isValidImportFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

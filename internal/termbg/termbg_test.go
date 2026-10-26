package termbg

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		reply string
		want  string
		ok    bool
	}{
		{"\x1b]11;rgb:1616/1616/1616\x1b\\", "#161616", true},
		{"\x1b]11;rgb:ffff/8000/0000\a", "#ff8000", true},
		{"rgb:22/6a/4d", "#226a4d", true},
		{"rgb:f/0/8", "#ff0088", true},
		{"rgb:fff/000/888", "#ff0088", true},
		{"", "", false},
		{"\x1b]11;?\x1b\\", "", false},
	}
	for _, tt := range tests {
		got, ok := Parse(tt.reply)
		if got != tt.want || ok != tt.ok {
			t.Fatalf("Parse(%q) = %q, %t; want %q, %t", tt.reply, got, ok, tt.want, tt.ok)
		}
	}
}

package script

import "testing"

func TestStripComments(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no comments",
			input: "Hello world.",
			want:  "Hello world.",
		},
		{
			name:  "inline comment",
			input: "Before <!-- comment --> after.",
			want:  "Before  after.",
		},
		{
			name:  "multiline comment",
			input: "Before\n<!-- multi\nline\ncomment -->\nafter.",
			want:  "Before\n\nafter.",
		},
		{
			name:  "multiple comments",
			input: "A <!-- one --> B <!-- two --> C",
			want:  "A  B  C",
		},
		{
			name:  "empty comment",
			input: "A <!----> B",
			want:  "A  B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripComments(tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

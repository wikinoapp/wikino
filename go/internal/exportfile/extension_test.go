package exportfile

import (
	"path"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestDeduper_UnsafeAttachmentExtensions(t *testing.T) {
	t.Parallel()
	tests := []struct{ name, want string }{
		{"file.txt:stream", "file.txt：stream"},
		{"file.a\\b", "file.a＼b"},
		{"file.txt ", "file.txt"},
		{"file.txt\u202e", "file.txt"},
		{"file.txt?", "file.txt？"},
		{"file." + strings.Repeat("界", 100), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := NewDeduper()
			ext := path.Ext(tt.name)
			title := strings.TrimSuffix(tt.name, ext)
			seen := map[string]bool{}
			for range 12 {
				got := d.Unique(title, ext)
				if len(seen) == 0 && tt.want != "" && got != tt.want {
					t.Errorf("name=%q, want %q", got, tt.want)
				}
				if len(got) > 255 || !utf8.ValidString(got) {
					t.Errorf("invalid length or UTF-8: %q (%d bytes)", got, len(got))
				}
				if strings.ContainsAny(got, "/\\:*?<>|\u202e") || strings.HasSuffix(got, " ") || strings.HasSuffix(got, ".") {
					t.Errorf("unsafe name: %q", got)
				}
				if seen[got] {
					t.Errorf("duplicate name: %q", got)
				}
				seen[got] = true
			}
		})
	}
}

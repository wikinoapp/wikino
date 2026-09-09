package markup

import (
	"reflect"
	"strings"
	"testing"
)

func TestScanAttachmentRefMatches_PreservesParsedDocument(t *testing.T) {
	t.Parallel()
	bodies := []string{
		"[![image](/attachments/01A)](/attachments/01A)",
		"[one][r] ![two][r]\n\n[r]: /attachments/01A",
		"<img src='/attachments/01A'> <a href=/attachments/01A>file</a>",
		"<div>\n<img src=\"/attachments/01A\">\n<img src=\"/attachments/01A\"></div>",
		"<!-- comment\n--><img src=\"/attachments/01A\"><img src=\"/attachments/01A\">",
		"[visible](/attachments/01A)\n\n> <!--\n\n<img src=\"/attachments/01A\">",
	}
	for _, body := range bodies {
		t.Run(body, func(t *testing.T) {
			t.Parallel()
			source, document, before := renderBody(body)
			first := scanAttachmentRefMatches(source, document, before, true)
			second := scanAttachmentRefMatches(source, document, before, true)
			if !reflect.DeepEqual(first, second) {
				t.Errorf("repeat scan changed matches: %v / %v", first, second)
			}
			after, err := renderSanitized(source, document)
			if err != nil {
				t.Fatal(err)
			}
			if after != before {
				t.Errorf("document changed: before=%q after=%q", before, after)
			}
			if string(source) != body {
				t.Errorf("source changed: %q", source)
			}
			if strings.Contains(after, "Wikino") {
				t.Errorf("marker leaked into output: %s", after)
			}
		})
	}
}

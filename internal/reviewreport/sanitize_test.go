package reviewreport

import (
	"strings"
	"testing"
)

func TestSanitizeFieldRemovesDiffLogsTranscriptsAndSecrets(t *testing.T) {
	raw := "before\n" +
		"```go\nfmt.Println(\"hidden\")\n```\n" +
		"diff --git a/a.go b/a.go\n" +
		"+token=supersecret\n" +
		"stdout: raw log output\n" +
		"transcript: user pasted sk-abcdefghijklmnop\n" +
		"api_key=sk-abcdefghijklmnop\n" +
		"after"

	got := sanitizeField(raw)
	for _, disallowed := range []string{"fmt.Println", "diff --git", "+token", "raw log output", "user pasted", "sk-abcdefghijklmnop"} {
		if strings.Contains(got, disallowed) {
			t.Fatalf("sanitized field leaked %q in %q", disallowed, got)
		}
	}
	for _, want := range []string{"\\[code block removed\\]", "\\[diff content removed\\]", "\\[log/transcript content removed\\]", "api\\_key=\\[REDACTED\\]", "after"} {
		if !strings.Contains(got, want) {
			t.Fatalf("sanitized field missing %q in %q", want, got)
		}
	}
}

func TestSanitizeFieldEscapesMarkdownAndTruncates(t *testing.T) {
	got := sanitizeShort("# [x] _secret_ `" + strings.Repeat("a", maxShortFieldRunes+20))
	for _, want := range []string{`\#`, `\[x\]`, `\_secret\_`, "\\`", "[truncated]"} {
		if !strings.Contains(got, want) {
			t.Fatalf("sanitized short field missing %q in %q", want, got)
		}
	}
}

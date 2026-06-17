package reviewreport

import (
	"regexp"
	"strings"
	"unicode"
)

const maxSanitizedTextLen = 500

var (
	singleLineAccessPattern     = `[A-Za-z_][A-Za-z0-9_]*(?:(?:\.[A-Za-z_][A-Za-z0-9_]*)|(?:\[[^]]+\]))*`
	singleLineAssignmentPattern = regexp.MustCompile(`^` + singleLineAccessPattern + `\s*(?:=|\+=|-=|\*=|/=|%=)\s*\S+`)
	singleLineCallPattern       = regexp.MustCompile(`\b` + singleLineAccessPattern + `\([^)]*\)`)
	singleLineIndexPattern      = regexp.MustCompile(`\b` + singleLineAccessPattern + `\[[^]]+\]`)
)

// SanitizeText keeps report fields on the allowlisted, user-facing path.
// Unsafe or transcript/log/diff/code-like content is replaced rather than copied.
func SanitizeText(raw string, missing string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return missing
	}
	lower := strings.ToLower(text)
	if looksSecretLike(lower) || looksDiffLike(text) || looksLogLike(lower) || looksTranscriptLike(lower) || looksCodeLike(text) {
		return ValueUnavailable
	}
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > maxSanitizedTextLen {
		text = text[:maxSanitizedTextLen]
		for len(text) > 0 && !unicode.IsSpace(rune(text[len(text)-1])) {
			text = text[:len(text)-1]
		}
		text = strings.TrimSpace(text) + "..."
	}
	return text
}

func SanitizeAppliedFixSummary(raw string) string {
	clean := SanitizeText(raw, AppliedFixSummaryDisplayMissing)
	if clean == ValueUnavailable {
		return AppliedFixSummaryDisplayOmitted
	}
	return clean
}

func looksSecretLike(lower string) bool {
	needles := []string{
		"-----begin ",
		"password=",
		"password:",
		"api_key",
		"apikey",
		"access_token",
		"secret",
		"sk-",
	}
	for _, needle := range needles {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return false
}

func looksDiffLike(text string) bool {
	return strings.Contains(text, "diff --git ") ||
		strings.Contains(text, "\n@@") ||
		strings.Contains(text, "\n--- ") ||
		strings.Contains(text, "\n+++ ") ||
		strings.HasPrefix(text, "@@")
}

func looksLogLike(lower string) bool {
	return strings.Contains(lower, "panic:") ||
		strings.Contains(lower, "stack trace") ||
		strings.Contains(lower, "goroutine ") ||
		strings.Contains(lower, "traceback (most recent call last)")
}

func looksTranscriptLike(lower string) bool {
	return strings.Contains(lower, "<assistant") ||
		strings.Contains(lower, "<user") ||
		strings.Contains(lower, "assistant:") ||
		strings.Contains(lower, "human:")
}

func looksCodeLike(text string) bool {
	lines := strings.Split(text, "\n")
	if len(lines) < 2 {
		return looksSingleLineCodeLike(strings.TrimSpace(text))
	}
	codeLines := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "func ") ||
			strings.HasPrefix(trimmed, "package ") ||
			strings.HasPrefix(trimmed, "import ") ||
			strings.HasPrefix(trimmed, "class ") ||
			strings.HasPrefix(trimmed, "def ") ||
			strings.HasSuffix(trimmed, " {") ||
			strings.HasPrefix(trimmed, "return") {
			codeLines++
		}
	}
	return codeLines >= 2
}

func looksSingleLineCodeLike(line string) bool {
	if line == "" {
		return false
	}
	lower := strings.ToLower(line)
	prefixes := []string{
		"const ",
		"def ",
		"func ",
		"function ",
		"if ",
		"import ",
		"let ",
		"package ",
		"return ",
		"var ",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(lower, prefix) && containsCodePunctuation(line) {
			return true
		}
	}
	return strings.Contains(line, ":=") ||
		strings.Contains(lower, " != nil") ||
		strings.Contains(lower, " == nil") ||
		strings.Contains(line, "=>") ||
		strings.Contains(line, "->") ||
		strings.Contains(line, "++") ||
		strings.Contains(line, "--") ||
		strings.Contains(line, "&&") ||
		strings.Contains(line, "||") ||
		strings.Contains(line, ";") ||
		singleLineAssignmentPattern.MatchString(line) ||
		singleLineCallPattern.MatchString(line) ||
		singleLineIndexPattern.MatchString(line) ||
		(strings.Contains(line, "{") && strings.Contains(line, "}") && (strings.Contains(lower, "return ") || strings.Contains(lower, "if ")))
}

func containsCodePunctuation(line string) bool {
	return strings.ContainsAny(line, "{}();=.")
}

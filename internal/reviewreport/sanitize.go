package reviewreport

import (
	"html"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	maxFieldRunes       = 800
	maxShortFieldRunes  = 180
	maxChangedFileRunes = 160
)

var (
	secretAssignmentPattern = regexp.MustCompile(`(?i)\b(api[_-]?key|token|secret|password|passwd)\b\s*[:=]\s*["']?[^"'\s]+`)
	secretValuePattern      = regexp.MustCompile(`\b(sk-[A-Za-z0-9_-]{12,}|gh[pousr]_[A-Za-z0-9_]{12,}|xox[baprs]-[A-Za-z0-9-]{12,})\b`)
	rawDiffPattern          = regexp.MustCompile(`(?m)^(diff --git .+|index [0-9a-f]+\.\.[0-9a-f]+.*|@@ .+ @@.*|[+-]{3} .+|[+-].*)$`)
)

func sanitizeField(value string) string {
	return sanitizeBounded(value, maxFieldRunes)
}

func sanitizeShort(value string) string {
	return sanitizeBounded(value, maxShortFieldRunes)
}

func sanitizeChangedFile(value string) string {
	clean := sanitizeBounded(value, maxChangedFileRunes)
	clean = strings.ReplaceAll(clean, "`", "")
	clean = strings.TrimSpace(clean)
	if clean == "." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "\n") {
		return ""
	}
	return clean
}

func sanitizeBounded(value string, limit int) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = stripCodeFences(value)
	value = rawDiffPattern.ReplaceAllString(value, "[diff content removed]")
	value = secretAssignmentPattern.ReplaceAllString(value, "$1=[REDACTED]")
	value = secretValuePattern.ReplaceAllString(value, "[REDACTED_SECRET]")
	lines := strings.Split(value, "\n")
	for i := range lines {
		lines[i] = strings.Join(strings.Fields(lines[i]), " ")
	}
	value = strings.TrimSpace(strings.Join(lines, "\n"))
	value = html.EscapeString(value)
	value = strings.NewReplacer(
		"\\", "\\\\",
		"`", "\\`",
		"*", "\\*",
		"_", "\\_",
		"[", "\\[",
		"]", "\\]",
		"#", "\\#",
	).Replace(value)
	return truncateRunes(value, limit)
}

func stripCodeFences(value string) string {
	if !strings.Contains(value, "```") {
		return value
	}
	parts := strings.Split(value, "```")
	var out []string
	for i, part := range parts {
		if i%2 == 1 {
			out = append(out, "[code block removed]")
			continue
		}
		out = append(out, part)
	}
	return strings.Join(out, "")
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 || utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit]) + " ... [truncated]"
}

func fallback(value, marker string) string {
	if strings.TrimSpace(value) == "" {
		return marker
	}
	return value
}

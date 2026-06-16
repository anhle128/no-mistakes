package reviewhandoff

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func Render(file HandoffFile) ([]byte, error) {
	var body bytes.Buffer
	body.WriteString("# Review handoff\n\n")
	renderSummary(&body, file.Summary)
	if len(file.Findings) > 0 {
		body.WriteString("\n## Findings\n")
		for i, finding := range file.Findings {
			response := responseForFinding(file.Responses, finding)
			renderFinding(&body, i+1, finding, response)
		}
	}
	if len(file.AuditEntries) > 0 {
		body.WriteString("\n## Prior Decisions\n")
		for _, entry := range file.AuditEntries {
			renderAuditEntry(&body, entry)
		}
	}
	if strings.TrimSpace(file.FinalState) != "" {
		body.WriteString("\n## Final State\n\n")
		body.WriteString(strings.TrimSpace(file.FinalState))
		body.WriteString("\n")
	}
	return renderWithBody(file.Metadata, body.Bytes())
}

func renderWithBody(meta Metadata, body []byte) ([]byte, error) {
	var out bytes.Buffer
	metaBytes, err := yaml.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshal review metadata: %w", err)
	}
	out.WriteString("---\n")
	out.Write(metaBytes)
	out.WriteString("---\n")
	out.Write(body)
	return out.Bytes(), nil
}

func renderSummary(b *bytes.Buffer, summary Summary) {
	fmt.Fprintf(b, "Total findings: %d\n", summary.Total)
	if len(summary.Severities) == 0 {
		return
	}
	keys := make([]string, 0, len(summary.Severities))
	for key := range summary.Severities {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	b.WriteString("\nSeverity counts:\n")
	for _, key := range keys {
		fmt.Fprintf(b, "- %s: %d\n", key, summary.Severities[key])
	}
}

func renderFinding(b *bytes.Buffer, index int, finding FindingEntry, response ResponseBlock) {
	fmt.Fprintf(b, "\n### %d. %s\n\n", index, finding.ID)
	if finding.Severity != "" {
		fmt.Fprintf(b, "- Severity: %s\n", finding.Severity)
	}
	if finding.File != "" {
		ref := finding.File
		if finding.Line > 0 {
			ref = fmt.Sprintf("%s:%d", finding.File, finding.Line)
		}
		fmt.Fprintf(b, "- Location: %s\n", ref)
	}
	b.WriteString("\n#### Issue\n\n")
	writeParagraph(b, finding.Issue)
	b.WriteString("\n#### Context\n\n")
	writeParagraph(b, finding.Context)
	b.WriteString("\n#### Recommendation\n\n")
	if len(finding.Recommendations) == 0 {
		b.WriteString("1. Review this finding and address the underlying issue.\n")
	} else {
		for i, rec := range finding.Recommendations {
			fmt.Fprintf(b, "%d. %s\n", i+1, strings.TrimSpace(rec))
		}
	}
	b.WriteString("\n#### User Answer\n\n")
	fmt.Fprintf(b, "```%s\n", FenceLanguage)
	block, _ := yaml.Marshal(response)
	b.Write(block)
	b.WriteString("```\n")
}

func writeParagraph(b *bytes.Buffer, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		b.WriteString("_No details provided._\n")
		return
	}
	b.WriteString(text)
	b.WriteString("\n")
}

func responseForFinding(responses []ResponseBlock, finding FindingEntry) ResponseBlock {
	for _, response := range responses {
		if response.ID == finding.ID {
			return response
		}
	}
	return ResponseBlock{ID: finding.ID, Action: finding.DefaultResponseAction}
}

func InitialResponses(entries []FindingEntry) []ResponseBlock {
	responses := make([]ResponseBlock, 0, len(entries))
	for _, entry := range entries {
		action := entry.DefaultResponseAction
		if !validResponseAction(action) {
			action = ActionAccept
		}
		responses = append(responses, ResponseBlock{ID: entry.ID, Action: action})
	}
	return responses
}

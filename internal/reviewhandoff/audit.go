package reviewhandoff

import (
	"bytes"
	"fmt"
	"strings"
)

func renderAuditEntry(b *bytes.Buffer, entry AuditEntry) {
	title := strings.TrimSpace(entry.FindingID)
	if title == "" {
		title = "review finding"
	}
	fmt.Fprintf(b, "\n### %s\n\n", title)
	writeAuditField(b, "Severity", entry.Severity)
	writeAuditField(b, "Issue", entry.Issue)
	writeAuditField(b, "Action", entry.Action)
	writeAuditField(b, "Solution", entry.Solution)
	writeAuditField(b, "Selection", entry.Selection)
	writeAuditField(b, "Fix summary", entry.FixSummary)
	writeAuditField(b, "Source", entry.Source)
	writeAuditField(b, "Processed", entry.ProcessedTime)
}

func writeAuditField(b *bytes.Buffer, label, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	fmt.Fprintf(b, "- %s: %s\n", label, value)
}

func AuditEntriesFromDecision(decision ProcessedDecision, entries []FindingEntry) []AuditEntry {
	selected := make(map[string]bool, len(decision.SelectedFindingIDs))
	for _, id := range decision.SelectedFindingIDs {
		selected[id] = true
	}
	audit := make([]AuditEntry, 0, len(entries)+len(decision.AddedFindings))
	for _, entry := range entries {
		action := ActionAccept
		selection := "not sent to fixer"
		solution := ""
		if entryAction := decision.Actions[entry.ID]; entryAction != "" {
			action = entryAction
		} else if decision.ExecutedAction == ProcessedSkip {
			action = ActionSkip
		}
		if selected[entry.ID] {
			action = ActionFix
			selection = "sent to fixer"
			if decision.Instructions != nil {
				solution = decision.Instructions[entry.ID]
			}
			if solution == "" {
				solution = defaultRecommendation(entry)
			}
		} else if action == ActionSkip {
			selection = "skipped by user"
		}
		audit = append(audit, AuditEntry{
			FindingID:     entry.ID,
			Severity:      entry.Severity,
			Issue:         entry.Issue,
			Action:        action,
			Solution:      solution,
			Selection:     selection,
			Source:        decision.Source,
			ProcessedTime: decision.ProcessedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	for _, added := range decision.AddedFindings {
		audit = append(audit, AuditEntry{
			FindingID:     added.ID,
			Severity:      added.Severity,
			Issue:         added.Description,
			Action:        ActionFix,
			Solution:      added.UserInstructions,
			Selection:     "user-added finding sent to fixer",
			Source:        decision.Source,
			ProcessedTime: decision.ProcessedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return audit
}

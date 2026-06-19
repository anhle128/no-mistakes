package reviewreport

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const maxFullDetailReportBytes = 256 * 1024

// RenderMarkdown renders a Review resolution snapshot into the local report
// artifact format.
func RenderMarkdown(snap Snapshot) string {
	counts := CountEntries(snap.Entries)
	var b strings.Builder
	b.WriteString("# Review Resolution Report\n\n")
	b.WriteString("Report Format Version: ")
	b.WriteString(reportVersion)
	b.WriteString("\n\n")
	b.WriteString("## Run Context\n\n")
	writeField(&b, "Run ID", snap.RunID)
	writeField(&b, "Repository identifier or path", fallback(sanitizeField(snap.RepoIdentifier), "not recorded"))
	writeField(&b, "Branch", sanitizeShort(snap.Branch))
	writeField(&b, "Base commit", sanitizeShort(snap.BaseSHA))
	writeField(&b, "Current/final head commit", sanitizeShort(snap.HeadSHA))
	writeField(&b, "Review step status", sanitizeShort(snap.ReviewStatus))
	writeField(&b, "Report lifecycle state", sanitizeShort(snap.ReportStatus))
	writeField(&b, "First generated timestamp", formatUnix(snap.FirstGenerated))
	writeField(&b, "Last refreshed timestamp", formatUnix(snap.LastRefreshed))
	if snap.FinalizedAt != nil {
		writeField(&b, "Finalized timestamp", formatUnix(*snap.FinalizedAt))
	} else {
		writeField(&b, "Finalized timestamp", "not finalized")
	}
	writeField(&b, "Local report path", sanitizeField(snap.ReportPath))

	b.WriteString("\n## Counts\n\n")
	writeField(&b, "Resolved", fmt.Sprintf("%d", counts.Resolved))
	writeField(&b, "Accepted Without Fix", fmt.Sprintf("%d", counts.Accepted))
	writeField(&b, "Informational / No Action Required", fmt.Sprintf("%d", counts.Informational))
	writeField(&b, "Still Open", fmt.Sprintf("%d", counts.StillOpen))
	writeField(&b, "Total Entries", fmt.Sprintf("%d", len(snap.Entries)))

	detailsTruncated := false
	writeSection(&b, "Resolved Issues", snap.Entries, OutcomeResolved, &detailsTruncated)
	writeSection(&b, "Accepted Without Fix", snap.Entries, OutcomeAccepted, &detailsTruncated)
	writeSection(&b, "Informational / No Action Required", snap.Entries, OutcomeInformational, &detailsTruncated)
	writeSection(&b, "Still Open Issues", snap.Entries, OutcomeStillOpen, &detailsTruncated)
	if detailsTruncated {
		b.WriteString("\n## Report Truncated\n\n")
		b.WriteString(fmt.Sprintf("Report exceeded %d bytes of full detail; later entries are represented by compact ID- and provenance-preserving stubs.\n", maxFullDetailReportBytes))
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

func writeSection(b *strings.Builder, title string, entries []Entry, outcome EntryOutcome, detailsTruncated *bool) {
	b.WriteString("\n## ")
	b.WriteString(title)
	b.WriteString("\n\n")
	filtered := make([]Entry, 0)
	for _, entry := range entries {
		if entry.Outcome == outcome {
			filtered = append(filtered, entry)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].FirstRound == filtered[j].FirstRound {
			return filtered[i].Finding.ID < filtered[j].Finding.ID
		}
		return filtered[i].FirstRound < filtered[j].FirstRound
	})
	if len(filtered) == 0 {
		b.WriteString("No issues in this category.\n")
		return
	}
	for _, entry := range filtered {
		rendered := renderEntry(entry)
		if b.Len()+len(rendered) <= maxFullDetailReportBytes {
			b.WriteString(rendered)
			continue
		}
		if detailsTruncated != nil {
			*detailsTruncated = true
		}
		writeEntryStub(b, entry)
	}
}

func renderEntry(entry Entry) string {
	var b strings.Builder
	writeEntry(&b, entry)
	return b.String()
}

func writeEntry(b *strings.Builder, entry Entry) {
	id := fallback(sanitizeShort(entry.Finding.ID), "unavailable in historical data")
	b.WriteString("### ")
	b.WriteString(id)
	b.WriteString("\n\n")
	writeField(b, "Finding ID", id)
	writeField(b, "Severity", fallback(sanitizeShort(entry.Finding.Severity), "unavailable in historical data"))
	location := "unavailable in historical data"
	if entry.Finding.File != "" {
		location = sanitizeChangedFile(entry.Finding.File)
		if entry.Finding.Line > 0 {
			location = fmt.Sprintf("%s:%d", location, entry.Finding.Line)
		}
	}
	writeField(b, "File and line", location)
	writeField(b, "Action", fallback(sanitizeShort(entry.Finding.Action), "auto-fix"))
	source := entry.Finding.Source
	if source == "" {
		source = "agent"
	}
	writeField(b, "Source", sanitizeShort(source))
	if entry.FirstRound > 0 {
		writeField(b, "Review round ID", fmt.Sprintf("%d", entry.FirstRound))
	} else {
		writeField(b, "Review round ID", "not recorded")
	}
	if entry.LastSeenRound > 0 && entry.LastSeenRound != entry.FirstRound {
		writeField(b, "Last seen Review round ID", fmt.Sprintf("%d", entry.LastSeenRound))
	}
	writeField(b, "Description", fallback(sanitizeField(entry.Finding.Description), "unavailable in historical data"))
	writeField(b, "Context", "unavailable in historical data")
	writeField(b, "Suggested/proposed fix", "unavailable in historical data")
	writeField(b, "Risk level", "unavailable in historical data")
	writeField(b, "Risk rationale", "unavailable in historical data")
	writeField(b, "User instructions", fallback(sanitizeField(entry.Finding.UserInstructions), "not recorded"))
	writeField(b, "Outcome", outcomeLabel(entry.Outcome))
	writeField(b, "Outcome evidence and provenance", fallback(sanitizeField(entry.OutcomeText), "not recorded"))
	writeField(b, "Selection source", fallback(sanitizeShort(entry.SelectionSource), "not recorded"))
	if entry.Decision != nil {
		writeField(b, "Decision action", sanitizeShort(entry.Decision.Action))
		writeField(b, "Decision actor/source", sanitizeShort(entry.Decision.ActorSource))
		writeField(b, "Decision timestamp", formatUnix(entry.Decision.CreatedAt))
		writeField(b, "Decision round ID", fallback(sanitizeShort(entry.Decision.RoundID), "not recorded"))
		writeField(b, "Decision reason", fallback(sanitizeField(entry.Decision.Reason), "reason unavailable"))
	} else {
		writeField(b, "Decision action", "not recorded")
		writeField(b, "Decision actor/source", "not recorded")
		writeField(b, "Decision timestamp", "not recorded")
		writeField(b, "Decision round ID", "not recorded")
		writeField(b, "Decision reason", "not recorded")
	}
	if entry.FixRound > 0 {
		writeField(b, "Fix round ID", fmt.Sprintf("%d", entry.FixRound))
	} else {
		writeField(b, "Fix round ID", "not recorded")
	}
	writeField(b, "Applied Solution Source", fallback(sanitizeField(entry.SolutionSource), "not recorded"))
	writeField(b, "Applied solution or attempted solution", fallback(sanitizeField(entry.AppliedSolution), "not recorded"))
	writeField(b, "Rationale", fallback(sanitizeField(entry.Rationale), "not recorded"))
	writeField(b, "Changed files", strings.Join(entry.ChangedFiles, ", "))
	if entry.FixCommitSHA != "" {
		writeField(b, "Fix commit SHA", entry.FixCommitSHA)
	} else {
		writeField(b, "Fix commit SHA", "not recorded")
	}
	writeField(b, "No-commit reason", fallback(sanitizeShort(entry.NoCommitReason), "not recorded"))
	writeField(b, "Verification text", fallback(sanitizeField(entry.Verification), "not recorded"))
	if entry.FollowupRound > 0 {
		writeField(b, "Follow-up round ID", fmt.Sprintf("%d", entry.FollowupRound))
	} else {
		writeField(b, "Follow-up round ID", "not recorded")
	}
	writeField(b, "Scope-equivalence note", fallback(sanitizeField(entry.ScopeNote), "not recorded"))
	writeField(b, "Verifier source", fallback(sanitizeField(entry.VerifierSource), "not recorded"))
	writeField(b, "Evidence reference", fallback(sanitizeField(entry.EvidenceReference), "not recorded"))
	writeField(b, "Evidence quality", fallback(sanitizeShort(entry.EvidenceQuality), "unavailable"))
	b.WriteString("\n")
}

func writeEntryStub(b *strings.Builder, entry Entry) {
	id := fallback(sanitizeShort(entry.Finding.ID), "unavailable in historical data")
	b.WriteString("### ")
	b.WriteString(id)
	b.WriteString("\n\n")
	writeField(b, "Finding ID", id)
	writeField(b, "Outcome", outcomeLabel(entry.Outcome))
	if entry.FirstRound > 0 {
		writeField(b, "Review round ID", fmt.Sprintf("%d", entry.FirstRound))
	} else {
		writeField(b, "Review round ID", "not recorded")
	}
	writeField(b, "Selection source", fallback(sanitizeShort(entry.SelectionSource), "not recorded"))
	if entry.Decision != nil {
		writeField(b, "Decision action", sanitizeShort(entry.Decision.Action))
		writeField(b, "Decision actor/source", sanitizeShort(entry.Decision.ActorSource))
		writeField(b, "Decision timestamp", formatUnix(entry.Decision.CreatedAt))
		writeField(b, "Decision round ID", fallback(sanitizeShort(entry.Decision.RoundID), "not recorded"))
		writeField(b, "Decision reason", fallback(sanitizeField(entry.Decision.Reason), "reason unavailable"))
	} else {
		writeField(b, "Decision action", "not recorded")
		writeField(b, "Decision actor/source", "not recorded")
		writeField(b, "Decision timestamp", "not recorded")
		writeField(b, "Decision round ID", "not recorded")
		writeField(b, "Decision reason", "not recorded")
	}
	if entry.FixRound > 0 {
		writeField(b, "Fix round ID", fmt.Sprintf("%d", entry.FixRound))
	} else {
		writeField(b, "Fix round ID", "not recorded")
	}
	if entry.FixCommitSHA != "" {
		writeField(b, "Fix commit SHA", entry.FixCommitSHA)
	} else {
		writeField(b, "Fix commit SHA", "not recorded")
	}
	writeField(b, "No-commit reason", fallback(sanitizeShort(entry.NoCommitReason), "not recorded"))
	writeField(b, "Verification text", fallback(sanitizeField(entry.Verification), "not recorded"))
	if entry.FollowupRound > 0 {
		writeField(b, "Follow-up round ID", fmt.Sprintf("%d", entry.FollowupRound))
	} else {
		writeField(b, "Follow-up round ID", "not recorded")
	}
	writeField(b, "Scope-equivalence note", fallback(sanitizeField(entry.ScopeNote), "not recorded"))
	writeField(b, "Verifier source", fallback(sanitizeField(entry.VerifierSource), "not recorded"))
	writeField(b, "Evidence reference", fallback(sanitizeField(entry.EvidenceReference), "not recorded"))
	writeField(b, "Evidence quality", fallback(sanitizeShort(entry.EvidenceQuality), "unavailable"))
	writeField(b, "Entry detail", "truncated because report detail budget was exceeded; finding retained in counts")
	b.WriteString("\n")
}

func writeField(b *strings.Builder, label, value string) {
	b.WriteString("- ")
	b.WriteString(label)
	b.WriteString(": ")
	b.WriteString(value)
	b.WriteString("\n")
}

func outcomeLabel(outcome EntryOutcome) string {
	switch outcome {
	case OutcomeResolved:
		return "Resolved"
	case OutcomeAccepted:
		return "Accepted Without Fix"
	case OutcomeInformational:
		return "Informational / No Action Required"
	default:
		return "Still Open"
	}
}

func formatUnix(ts int64) string {
	if ts <= 0 {
		return "not recorded"
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}

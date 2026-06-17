package reviewreport

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func RenderMarkdown(snapshot ReportSnapshot) string {
	var b strings.Builder
	b.WriteString("# Review Resolution Report\n\n")
	renderReportMetadata(&b, snapshot)
	renderPurpose(&b)
	renderRunContext(&b, snapshot)
	renderSummaryCounts(&b, snapshot.Counts)
	renderLatestOutcome(&b, snapshot.Latest)
	renderReviewFindings(&b, snapshot.Findings)
	renderFixAttempts(&b, snapshot.FixAttempts)
	renderRemainingRisks(&b, snapshot)
	renderSourceEvidence(&b, snapshot.SourceEvidence)
	renderGenerationNotes(&b, snapshot.Generation)
	return b.String()
}

func renderReportMetadata(b *strings.Builder, snapshot ReportSnapshot) {
	b.WriteString("## Report Metadata\n\n")
	writeLabel(b, "Contract version", ContractVersion)
	writeLabel(b, "Report path", valueOrUnavailable(snapshot.Metadata.Path))
	writeLabel(b, "Report status", valueOrUnavailable(snapshot.Metadata.Status))
	writeLabel(b, "Generation mode", valueOrUnavailable(snapshot.Metadata.GenerationMode))
	writeLabel(b, "Generated at", formatUnix(snapshot.Metadata.GeneratedAt))
	writeLabel(b, "Updated at", formatUnix(snapshot.Metadata.UpdatedAt))
	writeLabel(b, "Source snapshot at", formatUnix(snapshot.Metadata.SourceSnapshotAt))
	b.WriteString("\n")
}

func renderPurpose(b *strings.Builder) {
	b.WriteString("## Purpose\n\n")
	b.WriteString("This report records review findings, resolution decisions, fix attempts, applied fix summaries, and remaining risks for later human and agent review. It is a reporting reference only and does not change review, approval, auto-fix, push, PR, or CI behavior.\n\n")
}

func renderRunContext(b *strings.Builder, snapshot ReportSnapshot) {
	b.WriteString("## Run Context\n\n")
	writeLabel(b, "Run", valueOrUnavailable(snapshot.Run.ID))
	writeLabel(b, "Branch", valueOrUnavailable(snapshot.Run.Branch))
	writeLabel(b, "Base commit", valueOrUnavailable(snapshot.Run.BaseSHA))
	writeLabel(b, "Head commit", valueOrUnavailable(snapshot.Run.HeadSHA))
	writeLabel(b, "Run status", valueOrUnavailable(snapshot.Run.RunStatus))
	writeLabel(b, "Review status", valueOrUnavailable(snapshot.Run.ReviewStatus))
	writeLabel(b, "Safe intent summary", valueOrUnavailable(snapshot.Run.SafeIntentSummary))
	writeLabel(b, "PR", valueOrUnavailable(snapshot.Run.PRURL))
	b.WriteString("\n")
}

func renderSummaryCounts(b *strings.Builder, counts map[string]int) {
	b.WriteString("## Summary Counts\n\n")
	for _, key := range SummaryCountKeys {
		fmt.Fprintf(b, "- `%s`: %d\n", key, counts[key])
	}
	b.WriteString("\n")
}

func renderLatestOutcome(b *strings.Builder, latest LatestReviewOutcome) {
	b.WriteString("## Latest Review Outcome\n\n")
	writeLabel(b, "Latest outcome", valueOrUnavailable(latest.Outcome))
	writeLabel(b, "Evidence", valueOrUnavailable(latest.Evidence))
	writeLabel(b, "Risk", valueOrUnavailable(latest.Risk))
	writeLabel(b, "Rationale", valueOrUnavailable(latest.Rationale))
	b.WriteString("\n")
}

func renderReviewFindings(b *strings.Builder, findings []ReviewFinding) {
	b.WriteString("## Review Findings\n\n")
	if len(findings) == 0 {
		b.WriteString("No review findings recorded.\n\n")
		return
	}
	for i, finding := range findings {
		fmt.Fprintf(b, "### Finding %d\n\n", i+1)
		writeLabel(b, "Issue", valueOrUnavailable(finding.Issue))
		writeLabel(b, "Severity", valueOrUnavailable(finding.Severity))
		writeLabel(b, "Location", valueOrUnavailable(finding.Location))
		writeLabel(b, "Source", valueOrUnavailable(finding.Source))
		writeLabel(b, "Action type", valueOrUnavailable(finding.ActionType))
		writeLabel(b, "Context", valueOrUnavailable(finding.Context))
		writeLabel(b, "Recommendation", valueOrUnavailable(finding.Recommendation))
		writeLabel(b, "Selected for fix", yesNo(finding.SelectedForFix))
		writeLabel(b, "Resolution decision", valueOrUnavailable(finding.Decision))
		writeLabel(b, "Decision actor", valueOrUnavailable(finding.DecisionActor))
		writeLabel(b, "Decision evidence", valueOrUnavailable(finding.DecisionEvidence))
		writeLabel(b, "User instructions", valueOrUnavailable(finding.UserInstructions))
		b.WriteString("\n")
	}
}

func renderFixAttempts(b *strings.Builder, attempts []FixAttempt) {
	b.WriteString("## Fix Attempts\n\n")
	if len(attempts) == 0 {
		b.WriteString("No fix attempts recorded.\n\n")
		return
	}
	for _, attempt := range attempts {
		writeLabel(b, "Fix attempt", fmt.Sprintf("round %d", attempt.Round))
		writeLabel(b, "Selected findings", joinOrValue(attempt.SelectedFindings, ValueNotRecorded))
		writeLabel(b, "Selection source", valueOrUnavailable(attempt.SelectionSource))
		writeLabel(b, "User instructions", valueOrUnavailable(attempt.UserInstructions))
		writeLabel(b, "User-authored findings", joinOrValue(attempt.UserAuthored, ValueNotRecorded))
		writeLabel(b, "Applied fix", valueOrUnavailable(attempt.AppliedFix))
		writeLabel(b, "Verification", valueOrUnavailable(attempt.Verification))
		writeLabel(b, "Evidence", valueOrUnavailable(attempt.Evidence))
		b.WriteString("\n")
	}
}

func renderRemainingRisks(b *strings.Builder, snapshot ReportSnapshot) {
	b.WriteString("## Remaining Risks\n\n")
	var risks []string
	if snapshot.Counts[CountStillOpen] > 0 {
		risks = append(risks, fmt.Sprintf("%d finding(s) still open", snapshot.Counts[CountStillOpen]))
	}
	if snapshot.Counts[CountUnavailable] > 0 {
		risks = append(risks, fmt.Sprintf("%d finding(s) unavailable", snapshot.Counts[CountUnavailable]))
	}
	if snapshot.Counts[CountDecisionNotRecorded] > 0 {
		risks = append(risks, fmt.Sprintf("%d finding decision(s) not recorded", snapshot.Counts[CountDecisionNotRecorded]))
	}
	switch snapshot.Latest.Outcome {
	case LatestOutcomeReviewDataInconsistent, LatestOutcomeReviewResolutionIncomplete, LatestOutcomeFinalFindingsUnavailable, LatestOutcomeFinalFindingsUnreadable:
		risks = append(risks, snapshot.Latest.Outcome)
	}
	if len(risks) == 0 {
		b.WriteString("None recorded.\n\n")
		return
	}
	for _, risk := range risks {
		fmt.Fprintf(b, "- %s\n", risk)
	}
	b.WriteString("\n")
}

func renderSourceEvidence(b *strings.Builder, source SourceEvidence) {
	b.WriteString("## Source Evidence\n\n")
	writeLabel(b, "Review step result", valueOrUnavailable(source.ReviewStepResultID))
	writeLabel(b, "Included rounds", joinOrValue(source.IncludedRounds, ValueUnavailable))
	writeLabel(b, "Latest review round", valueOrUnavailable(source.LatestReviewRound))
	writeLabel(b, "Latest fix round", valueOrUnavailable(source.LatestFixRound))
	writeLabel(b, "Source snapshot at", formatUnix(source.SourceSnapshotAt))
	writeLabel(b, "Integrity status", valueOrUnavailable(source.IntegrityStatus))
	if len(source.Diagnostics) > 0 {
		writeLabel(b, "Diagnostics", joinOrValue(source.Diagnostics, ValueUnavailable))
	}
	b.WriteString("\n")
}

func renderGenerationNotes(b *strings.Builder, notes GenerationNotes) {
	b.WriteString("## Generation Notes\n\n")
	if len(notes.Warnings) == 0 {
		b.WriteString("No generation warnings\n")
		return
	}
	for _, warning := range notes.Warnings {
		fmt.Fprintf(b, "- %s\n", SanitizeText(warning, ValueUnavailable))
	}
}

func writeLabel(b *strings.Builder, label string, value string) {
	fmt.Fprintf(b, "- `%s`: %s\n", label, valueOrUnavailable(value))
}

func valueOrUnavailable(value string) string {
	if strings.TrimSpace(value) == "" {
		return ValueUnavailable
	}
	return value
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func joinOrValue(values []string, fallback string) string {
	if len(values) == 0 {
		return fallback
	}
	clean := append([]string(nil), values...)
	sort.Strings(clean)
	return strings.Join(clean, ", ")
}

func formatUnix(ts int64) string {
	if ts <= 0 {
		return ValueUnavailable
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}

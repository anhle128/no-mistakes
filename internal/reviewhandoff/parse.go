package reviewhandoff

import (
	"fmt"
	"strings"
)

const fenceTag = "no-mistakes-review-response"

func Parse(content []byte) (Metadata, map[string]Response, error) {
	meta, body, err := parseMetadata(string(content))
	if err != nil {
		return Metadata{}, nil, err
	}
	responses, err := parseResponses(body)
	if err != nil {
		return Metadata{}, nil, err
	}
	return meta, responses, nil
}

func parseMetadata(raw string) (Metadata, string, error) {
	if !strings.HasPrefix(raw, "---\n") {
		return Metadata{}, "", fmt.Errorf("review handoff front matter is missing")
	}
	end := strings.Index(raw[4:], "\n---")
	if end < 0 {
		return Metadata{}, "", fmt.Errorf("review handoff front matter is not closed")
	}
	block := raw[4 : 4+end]
	body := raw[4+end+len("\n---"):]
	values := map[string]string{}
	for _, line := range strings.Split(block, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return Metadata{}, "", fmt.Errorf("malformed metadata line %q", line)
		}
		key = strings.TrimSpace(key)
		if _, exists := values[key]; exists {
			return Metadata{}, "", fmt.Errorf("duplicate metadata field %q", key)
		}
		values[key] = strings.TrimSpace(value)
	}
	meta := Metadata{
		Marker:          values["no_mistakes_review_handoff"],
		RunID:           values["run_id"],
		Step:            values["step"],
		Status:          values["status"],
		Branch:          values["branch"],
		ReviewCycleID:   values["review_cycle_id"],
		FindingDigest:   values["finding_digest"],
		ReviewFile:      values["review_file"],
		ProcessedAction: values["processed_action"],
		ProcessedAt:     values["processed_at"],
	}
	for _, required := range []struct {
		name  string
		value string
	}{
		{"no_mistakes_review_handoff", meta.Marker},
		{"run_id", meta.RunID},
		{"step", meta.Step},
		{"status", meta.Status},
		{"branch", meta.Branch},
		{"review_cycle_id", meta.ReviewCycleID},
		{"finding_digest", meta.FindingDigest},
		{"review_file", meta.ReviewFile},
		{"processed_action", meta.ProcessedAction},
	} {
		if required.value == "" {
			return Metadata{}, "", fmt.Errorf("metadata field %q is required", required.name)
		}
	}
	if meta.Marker != "v1" {
		return Metadata{}, "", fmt.Errorf("unsupported review handoff marker %q", meta.Marker)
	}
	return meta, body, nil
}

func parseResponses(body string) (map[string]Response, error) {
	lines := strings.Split(body, "\n")
	responses := map[string]Response{}
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if !strings.HasPrefix(line, "```") {
			continue
		}
		info := strings.TrimSpace(strings.TrimPrefix(line, "```"))
		if !strings.HasPrefix(info, fenceTag) {
			continue
		}
		parts := strings.Fields(info)
		if len(parts) != 2 || parts[0] != fenceTag {
			return nil, fmt.Errorf("response fence must be exactly %q plus one finding id", fenceTag)
		}
		id := parts[1]
		if _, exists := responses[id]; exists {
			return nil, fmt.Errorf("duplicate response block for finding %s", id)
		}
		action := ""
		solution := ""
		seen := map[string]bool{}
		closed := false
		for i++; i < len(lines); i++ {
			current := lines[i]
			if strings.HasPrefix(current, "```") {
				if strings.TrimSpace(current) != "```" {
					return nil, fmt.Errorf("nested response fence in finding %s", id)
				}
				closed = true
				break
			}
			key, value, ok := strings.Cut(current, ":")
			if !ok {
				return nil, fmt.Errorf("malformed response line for finding %s", id)
			}
			key = strings.TrimSpace(key)
			if key != "action" && key != "solution" {
				return nil, fmt.Errorf("unknown response field %q for finding %s", key, id)
			}
			if seen[key] {
				return nil, fmt.Errorf("duplicate response field %q for finding %s", key, id)
			}
			seen[key] = true
			value = strings.TrimSpace(value)
			if key == "action" {
				action = value
			} else {
				solution = value
			}
		}
		if !closed {
			return nil, fmt.Errorf("response block for finding %s is not closed", id)
		}
		switch action {
		case "fix", "accept", "skip":
		default:
			return nil, fmt.Errorf("unsupported action %q for finding %s", action, id)
		}
		if !seen["action"] || !seen["solution"] {
			return nil, fmt.Errorf("response block for finding %s must include action and solution", id)
		}
		responses[id] = Response{FindingID: id, Action: action, Solution: solution}
	}
	return responses, nil
}

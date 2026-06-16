package reviewhandoff

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func ParseFrontMatter(data []byte) (Metadata, []byte, error) {
	text := string(data)
	if !strings.HasPrefix(text, "---\n") {
		return Metadata{}, nil, fmt.Errorf("missing YAML front matter")
	}
	rest := text[len("---\n"):]
	idx := strings.Index(rest, "\n---\n")
	if idx < 0 {
		return Metadata{}, nil, fmt.Errorf("unterminated YAML front matter")
	}
	rawMeta := rest[:idx]
	body := []byte(rest[idx+len("\n---\n"):])
	var meta Metadata
	if err := yaml.Unmarshal([]byte(rawMeta), &meta); err != nil {
		return Metadata{}, nil, fmt.Errorf("parse YAML front matter: %w", err)
	}
	return meta, body, nil
}

func ParseResponseBlocks(data []byte) ([]ResponseBlock, error) {
	lines := bytes.Split(data, []byte("\n"))
	var blocks []ResponseBlock
	for i := 0; i < len(lines); i++ {
		if strings.TrimSpace(string(lines[i])) != "```"+FenceLanguage {
			continue
		}
		start := i + 1
		end := -1
		for j := start; j < len(lines); j++ {
			if strings.TrimSpace(string(lines[j])) == "```" {
				end = j
				break
			}
		}
		if end < 0 {
			return nil, fmt.Errorf("unterminated %s block", FenceLanguage)
		}
		raw := bytes.Join(lines[start:end], []byte("\n"))
		var block ResponseBlock
		if err := yaml.Unmarshal(raw, &block); err != nil {
			return nil, fmt.Errorf("parse response block: %w", err)
		}
		blocks = append(blocks, block)
		i = end
	}
	return blocks, nil
}

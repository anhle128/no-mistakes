package reviewhandoff

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type PathInput struct {
	WorkDir string
	Branch  string
	RunID   string
	BaseSHA string
	HeadSHA string
}

func ResolvePath(ctx context.Context, input PathInput) (string, error) {
	name := fmt.Sprintf("review-issues-%s.md", shortID(input.RunID))
	if anchor := singleAnchor(input.WorkDir, changedFiles(ctx, input.WorkDir, "status", "--porcelain=v1", "--untracked-files=all")); anchor != "" {
		return filepath.ToSlash(filepath.Join(filepath.Dir(anchor), name)), nil
	}
	if input.BaseSHA != "" && input.HeadSHA != "" {
		if anchor := singleAnchor(input.WorkDir, changedFiles(ctx, input.WorkDir, "diff", "--name-only", input.BaseSHA, input.HeadSHA)); anchor != "" {
			return filepath.ToSlash(filepath.Join(filepath.Dir(anchor), name)), nil
		}
	}
	return filepath.ToSlash(filepath.Join(".no-mistakes", "issues", BranchSlug(input.Branch), name)), nil
}

func SafeJoin(root, rel string) (string, error) {
	return safeJoin(root, rel, false)
}

func SafeJoinForWrite(root, rel string) (string, error) {
	return safeJoin(root, rel, true)
}

func safeJoin(root, rel string, createParent bool) (string, error) {
	if root == "" {
		return "", fmt.Errorf("root is required")
	}
	if rel == "" {
		return "", fmt.Errorf("relative path is required")
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("path must be relative: %s", rel)
	}
	cleanRel := filepath.Clean(filepath.FromSlash(rel))
	if cleanRel == "." || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) || cleanRel == ".." {
		return "", fmt.Errorf("path escapes root: %s", rel)
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if createParent {
		if err := os.MkdirAll(rootAbs, 0o755); err != nil {
			return "", err
		}
	}
	target := filepath.Join(rootAbs, cleanRel)
	parent := filepath.Dir(target)
	evalRoot, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", err
	}
	existingParent := parent
	var missing []string
	for {
		if info, err := os.Lstat(existingParent); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				targetInfo, err := os.Stat(existingParent)
				if err != nil {
					return "", err
				}
				if !targetInfo.IsDir() {
					return "", fmt.Errorf("path parent is not a directory: %s", rel)
				}
				break
			}
			if !info.IsDir() {
				return "", fmt.Errorf("path parent is not a directory: %s", rel)
			}
			break
		} else if os.IsNotExist(err) {
			missing = append(missing, filepath.Base(existingParent))
			next := filepath.Dir(existingParent)
			if next == existingParent {
				return "", err
			}
			existingParent = next
			continue
		} else {
			return "", err
		}
	}
	evalExistingParent, err := filepath.EvalSymlinks(existingParent)
	if err != nil {
		return "", err
	}
	relParent, err := filepath.Rel(evalRoot, evalExistingParent)
	if err != nil || relParent == ".." || strings.HasPrefix(relParent, ".."+string(filepath.Separator)) || filepath.IsAbs(relParent) {
		return "", fmt.Errorf("path escapes root: %s", rel)
	}
	if createParent {
		for i := len(missing) - 1; i >= 0; i-- {
			existingParent = filepath.Join(existingParent, missing[i])
			if err := os.Mkdir(existingParent, 0o755); err != nil && !os.IsExist(err) {
				return "", err
			}
			if err := validateInsideRoot(evalRoot, existingParent, rel); err != nil {
				return "", err
			}
		}
		if err := validateInsideRoot(evalRoot, parent, rel); err != nil {
			return "", err
		}
	}
	return target, nil
}

func validateInsideRoot(evalRoot, path, originalRel string) error {
	evalPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	relPath, err := filepath.Rel(evalRoot, evalPath)
	if err != nil || relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) || filepath.IsAbs(relPath) {
		return fmt.Errorf("path escapes root: %s", originalRel)
	}
	return nil
}

func BranchSlug(branch string) string {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return "unknown"
	}
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(branch) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "unknown"
	}
	return out
}

func shortID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) > 8 {
		return id[:8]
	}
	if id == "" {
		return "unknown"
	}
	return id
}

func changedFiles(ctx context.Context, workDir string, args ...string) []string {
	out, err := exec.CommandContext(ctx, "git", append([]string{"-C", workDir}, args...)...).Output()
	if err != nil {
		return nil
	}
	statusOutput := len(args) > 0 && args[0] == "status"
	var files []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if statusOutput && len(line) >= 4 {
			line = strings.TrimSpace(line[3:])
			if idx := strings.LastIndex(line, " -> "); idx >= 0 {
				line = line[idx+4:]
			}
			line = unquoteStatusPath(line)
		} else {
			line = strings.TrimSpace(line)
		}
		files = append(files, filepath.ToSlash(line))
	}
	return files
}

func unquoteStatusPath(path string) string {
	path = strings.TrimSpace(path)
	if len(path) < 2 || path[0] != '"' || path[len(path)-1] != '"' {
		return path
	}
	unquoted, err := strconv.Unquote(path)
	if err != nil {
		return path
	}
	return unquoted
}

func singleAnchor(workDir string, files []string) string {
	var anchors []string
	for _, file := range files {
		base := filepath.Base(file)
		if base != "plan.md" && base != "tasks.md" {
			continue
		}
		if !regularAnchor(workDir, file) {
			continue
		}
		anchors = append(anchors, file)
	}
	if len(anchors) != 1 {
		return ""
	}
	return anchors[0]
}

func regularAnchor(workDir, rel string) bool {
	path, err := SafeJoin(workDir, rel)
	if err != nil {
		return false
	}
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

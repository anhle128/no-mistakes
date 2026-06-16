package reviewhandoff

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

var eligibleAnchorNames = map[string]bool{
	"plan.md":  true,
	"task.md":  true,
	"tasks.md": true,
}

type PathResolveInput struct {
	CheckoutDir             string
	RunID                   string
	Branch                  string
	ExistingReviewFilePaths []string
	UncommittedChangedPaths []string
	ReviewedChangedPaths    []string
}

type PathResult struct {
	AbsPath       string
	RelPath       string
	Source        string
	AnchorRelPath string
}

func ResolvePath(input PathResolveInput) (PathResult, error) {
	checkout, err := cleanExistingDir(input.CheckoutDir)
	if err != nil {
		return PathResult{}, err
	}
	name := FileName(input.RunID)
	if existing, ok, err := singleSafeExistingReviewFile(checkout, name, input.ExistingReviewFilePaths); err != nil {
		return PathResult{}, err
	} else if ok {
		return resultForPath(checkout, existing, "existing", "")
	}
	if dir, anchor, ok, err := singleAnchorDir(checkout, input.UncommittedChangedPaths); err != nil {
		return PathResult{}, err
	} else if ok {
		return resultForPath(checkout, filepath.Join(dir, name), "uncommitted_anchor", anchor)
	}
	if dir, anchor, ok, err := singleAnchorDir(checkout, input.ReviewedChangedPaths); err != nil {
		return PathResult{}, err
	} else if ok {
		return resultForPath(checkout, filepath.Join(dir, name), "reviewed_anchor", anchor)
	}
	fallback := filepath.Join(checkout, ".no-mistakes", "issues", BranchSlug(input.Branch), name)
	return resultForPath(checkout, fallback, "fallback", "")
}

func FindExistingReviewFiles(checkoutDir, runID string) ([]string, error) {
	checkout, err := cleanExistingDir(checkoutDir)
	if err != nil {
		return nil, err
	}
	name := FileName(runID)
	var matches []string
	err = filepath.WalkDir(checkout, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Base(path) != name {
			return nil
		}
		rel, err := filepath.Rel(checkout, path)
		if err != nil {
			return nil
		}
		matches = append(matches, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("find review files: %w", err)
	}
	return matches, nil
}

func SafeJoin(checkoutDir, rel string) (string, error) {
	checkout, err := cleanExistingDir(checkoutDir)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("path must be relative: %s", rel)
	}
	cleanRel := filepath.Clean(filepath.FromSlash(rel))
	if cleanRel == "." || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) || cleanRel == ".." {
		return "", fmt.Errorf("path escapes checkout: %s", rel)
	}
	for _, part := range strings.Split(cleanRel, string(filepath.Separator)) {
		if part == ".git" {
			return "", fmt.Errorf("path enters .git: %s", rel)
		}
	}
	abs := filepath.Join(checkout, cleanRel)
	if err := ensurePathInside(checkout, abs); err != nil {
		return "", err
	}
	if err := ensureNoSymlinkEscape(checkout, abs); err != nil {
		return "", err
	}
	return abs, nil
}

func BranchSlug(branch string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(branch)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "branch"
	}
	if len(slug) > 80 {
		slug = strings.Trim(slug[:80], "-")
	}
	if slug == "" {
		return "branch"
	}
	return slug
}

func IsEligibleAnchor(path string) bool {
	if filepath.IsAbs(path) {
		return false
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return false
	}
	for _, part := range strings.Split(clean, string(filepath.Separator)) {
		if part == ".git" {
			return false
		}
	}
	return eligibleAnchorNames[strings.ToLower(filepath.Base(clean))]
}

func cleanExistingDir(dir string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("checkout dir is required")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func singleSafeExistingReviewFile(checkout, filename string, paths []string) (string, bool, error) {
	var matches []string
	for _, candidate := range paths {
		if filepath.Base(candidate) != filename {
			continue
		}
		abs, err := SafeJoin(checkout, candidate)
		if err != nil {
			return "", false, err
		}
		matches = append(matches, abs)
	}
	if len(matches) == 1 {
		return matches[0], true, nil
	}
	return "", false, nil
}

func singleAnchorDir(checkout string, changed []string) (string, string, bool, error) {
	type candidate struct {
		dir string
		rel string
	}
	var anchors []candidate
	seen := map[string]bool{}
	for _, path := range changed {
		if !IsEligibleAnchor(path) {
			continue
		}
		abs, err := SafeJoin(checkout, path)
		if err != nil {
			return "", "", false, err
		}
		key := filepath.Clean(abs)
		if seen[key] {
			continue
		}
		seen[key] = true
		rel, _ := filepath.Rel(checkout, abs)
		anchors = append(anchors, candidate{dir: filepath.Dir(abs), rel: filepath.ToSlash(rel)})
	}
	if len(anchors) == 1 {
		return anchors[0].dir, anchors[0].rel, true, nil
	}
	return "", "", false, nil
}

func resultForPath(checkout, abs, source, anchor string) (PathResult, error) {
	if err := ensurePathInside(checkout, abs); err != nil {
		return PathResult{}, err
	}
	if err := ensureNoSymlinkEscape(checkout, abs); err != nil {
		return PathResult{}, err
	}
	rel, err := filepath.Rel(checkout, abs)
	if err != nil {
		return PathResult{}, err
	}
	return PathResult{AbsPath: abs, RelPath: filepath.ToSlash(rel), Source: source, AnchorRelPath: anchor}, nil
}

func ensureNoSymlinkEscape(checkout, abs string) error {
	if info, err := os.Lstat(abs); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("path is a symlink: %s", abs)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	ancestor, err := nearestExistingAncestor(abs)
	if err != nil {
		return err
	}
	resolved, err := filepath.EvalSymlinks(ancestor)
	if err != nil {
		return err
	}
	return ensurePathInside(checkout, resolved)
}

func nearestExistingAncestor(path string) (string, error) {
	dir := filepath.Dir(path)
	for {
		info, err := os.Stat(dir)
		if err == nil {
			if !info.IsDir() {
				return "", fmt.Errorf("path ancestor is not a directory: %s", dir)
			}
			return dir, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		next := filepath.Dir(dir)
		if next == dir {
			return "", fmt.Errorf("no existing ancestor for path: %s", path)
		}
		dir = next
	}
}

func ensurePathInside(root, path string) error {
	rootClean := filepath.Clean(root)
	pathClean := filepath.Clean(path)
	rel, err := filepath.Rel(rootClean, pathClean)
	if err != nil {
		return err
	}
	if rel == "." || rel == "" {
		return nil
	}
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
		return fmt.Errorf("path escapes checkout: %s", path)
	}
	return nil
}

package git

import (
	"context"
	"path/filepath"
	"sort"
	"testing"
)

// TestStatusChangedPaths_ModifiedTrackedFileFirst guards the porcelain parse
// against leading-byte corruption. A worktree-modified tracked file is reported
// with a leading status space (" M path"); if the whole status output is
// whitespace-trimmed before parsing, the first such record loses a character.
func TestStatusChangedPaths_ModifiedTrackedFileFirst(t *testing.T) {
	dir := initTestRepo(t)
	ctx := context.Background()

	// README.md is already tracked and committed. Modifying it (unstaged) makes
	// it a " M README.md" record; an untracked file that sorts after it keeps
	// README.md as the lexicographically first status record.
	writeFile(t, filepath.Join(dir, "README.md"), "# changed\n")
	writeFile(t, filepath.Join(dir, "znew.txt"), "new\n")

	paths, err := StatusChangedPaths(ctx, dir)
	if err != nil {
		t.Fatalf("StatusChangedPaths: %v", err)
	}
	got := map[string]bool{}
	for _, p := range paths {
		got[p] = true
	}
	if !got["README.md"] {
		t.Fatalf("expected uncorrupted README.md, got %v", paths)
	}
	if !got["znew.txt"] {
		t.Fatalf("expected znew.txt, got %v", paths)
	}
}

// TestStatusChangedPaths_Rename returns the destination path for a rename and
// discards the trailing source record git emits under -z.
func TestStatusChangedPaths_Rename(t *testing.T) {
	dir := initTestRepo(t)
	ctx := context.Background()

	writeFile(t, filepath.Join(dir, "old.txt"), "content\n")
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "add old")
	run(t, dir, "git", "mv", "old.txt", "new.txt")

	paths, err := StatusChangedPaths(ctx, dir)
	if err != nil {
		t.Fatalf("StatusChangedPaths: %v", err)
	}
	sort.Strings(paths)
	if len(paths) != 1 || paths[0] != "new.txt" {
		t.Fatalf("expected [new.txt], got %v", paths)
	}
}

// TestStatusChangedPaths_NonASCII keeps non-ASCII filenames verbatim instead of
// the octal-escaped quoted form git would emit with core.quotepath enabled.
func TestStatusChangedPaths_NonASCII(t *testing.T) {
	dir := initTestRepo(t)
	ctx := context.Background()
	run(t, dir, "git", "config", "core.quotepath", "true")

	writeFile(t, filepath.Join(dir, "café.md"), "x\n")

	paths, err := StatusChangedPaths(ctx, dir)
	if err != nil {
		t.Fatalf("StatusChangedPaths: %v", err)
	}
	if len(paths) != 1 || paths[0] != "café.md" {
		t.Fatalf("expected [café.md], got %v", paths)
	}
}

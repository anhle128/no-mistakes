package cli

import "testing"

func TestRootCommandAcceptsCurrentWorktreeAndYoloFlags(t *testing.T) {
	cmd := newRootCmd()

	if err := cmd.Flags().Parse([]string{"--no-worktree", "--yolo", "--yes"}); err != nil {
		t.Fatalf("root flags should parse together: %v", err)
	}
	for _, name := range []string{"no-worktree", "yolo", "yes"} {
		flag := cmd.Flags().Lookup(name)
		if flag == nil {
			t.Fatalf("missing root flag %q", name)
		}
		if flag.Value.String() != "true" {
			t.Fatalf("root flag %q = %q, want true", name, flag.Value.String())
		}
	}
}

func TestAxiRunAcceptsCurrentWorktreeAndYoloFlags(t *testing.T) {
	cmd := newAxiRunCmd()

	if err := cmd.Flags().Parse([]string{"--intent", "ship it", "--no-worktree", "--yolo", "--yes", "--skip", "lint"}); err != nil {
		t.Fatalf("axi run flags should parse together: %v", err)
	}
	for _, name := range []string{"no-worktree", "yolo", "yes"} {
		flag := cmd.Flags().Lookup(name)
		if flag == nil {
			t.Fatalf("missing axi run flag %q", name)
		}
		if flag.Value.String() != "true" {
			t.Fatalf("axi run flag %q = %q, want true", name, flag.Value.String())
		}
	}
	if got := cmd.Flags().Lookup("intent").Value.String(); got != "ship it" {
		t.Fatalf("intent = %q, want ship it", got)
	}
	if got := cmd.Flags().Lookup("skip").Value.String(); got != "lint" {
		t.Fatalf("skip = %q, want lint", got)
	}
}

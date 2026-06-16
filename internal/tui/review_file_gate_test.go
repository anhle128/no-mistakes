package tui

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kunchenguid/no-mistakes/internal/ipc"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

func TestReviewFileGateViewReplacesLegacyApprovalControls(t *testing.T) {
	configureTUIColors()
	run := testRun()
	run.Steps[0].Status = types.StepStatusAwaitingApproval
	run.Steps[0].FindingsJSON = ptr(`{"findings":[{"id":"review-1","severity":"error","description":"bug"},{"id":"review-2","severity":"warning","description":"warn one"},{"id":"review-3","severity":"warning","description":"warn two"}],"summary":"3 issues"}`)
	run.Steps[0].ReviewFilePath = ptr(".no-mistakes/issues/test-02/review-issues-01KV8J0W.md")
	run.Steps[0].ReviewFileAbsPath = ptr("/tmp/no-mistakes/worktrees/repo/run/.no-mistakes/issues/test-02/review-issues-01KV8J0W.md")
	m := NewModel("", nil, run)
	m.width = 80
	m.height = 50

	view := stripANSI(m.View())
	for _, want := range []string{
		"Review awaiting review file:",
		"p process",
		"c cancel",
		"Review File",
		"File: .no-mistakes/issues/test-02/review-issues-01KV8J0W.md",
		"Location: /tmp/no-mistakes/worktrees/repo/run",
		"Findings: (3) - E(1) - W(2)",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected %q in review file gate view:\n%s", want, view)
		}
	}
	for _, legacy := range []string{"a approve", "f fix", "s skip", "Findings -"} {
		if strings.Contains(view, legacy) {
			t.Fatalf("did not expect legacy control %q in review file gate view:\n%s", legacy, view)
		}
	}
}

func TestReviewFileGateProcessKeyCallsProcessReview(t *testing.T) {
	sock := testSocketPath(t)
	srv := startTestIPCServer(t, sock)
	captured := make(chan ipc.ProcessReviewParams, 1)
	srv.Handle(ipc.MethodProcessReview, func(_ context.Context, raw json.RawMessage) (interface{}, error) {
		var params ipc.ProcessReviewParams
		if err := json.Unmarshal(raw, &params); err != nil {
			return nil, err
		}
		captured <- params
		return &ipc.ProcessReviewResult{OK: true}, nil
	})
	client, err := ipc.Dial(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	run := testRun()
	run.Steps[0].Status = types.StepStatusAwaitingApproval
	run.Steps[0].ReviewFilePath = ptr("review-issues-run-001.md")
	m := NewModel(sock, client, run)

	updated, ignored := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if ignored != nil {
		t.Fatal("expected legacy approve key to be ignored in review file gate")
	}
	m = updated.(Model)

	updated, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if cmd == nil {
		t.Fatal("expected process review command")
	}
	_ = updated
	if msg := cmd(); msg != nil {
		t.Fatalf("process review command returned unexpected message: %#v", msg)
	}

	select {
	case params := <-captured:
		if params.RunID != run.ID || params.Step != types.StepReview {
			t.Fatalf("process params = %+v, want run %s step %s", params, run.ID, types.StepReview)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("process review RPC was not called")
	}
}

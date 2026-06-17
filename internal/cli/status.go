package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/kunchenguid/no-mistakes/internal/daemon"
	"github.com/kunchenguid/no-mistakes/internal/types"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show status of the current repository",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return trackCommandStatus("status", func() (string, error) {
				p, d, err := openResources()
				if err != nil {
					return "", err
				}
				defer d.Close()

				w := cmd.OutOrStdout()

				// Look up repo from current directory.
				repo, err := findRepo(d)
				if err != nil {
					fmt.Fprintln(w, err)
					return "error", nil
				}

				fmt.Fprintf(w, "  %s  %s\n", sDim.Render("  repo:"), repo.WorkingPath)
				fmt.Fprintf(w, "  %s  %s\n", sDim.Render("remote:"), repo.UpstreamURL)
				fmt.Fprintf(w, "  %s  %s\n", sDim.Render("  gate:"), p.RepoDir(repo.ID))

				// Check daemon status.
				alive, _ := daemon.IsRunning(p)
				if alive {
					fmt.Fprintf(w, "  %s  %s %s\n", sDim.Render("daemon:"), sGreen.Render("●"), "running")
				} else {
					fmt.Fprintf(w, "  %s  %s %s\n", sDim.Render("daemon:"), sDim.Render("○"), "stopped")
				}

				// Check for active run.
				activeRun, err := d.GetActiveRun(repo.ID, "")
				if err != nil {
					return "", fmt.Errorf("check active run: %w", err)
				}
				if activeRun != nil {
					fmt.Fprintln(w)
					fmt.Fprintf(w, "  %s\n", sCyan.Render("Active run"))
					sha := activeRun.HeadSHA[:minLen(len(activeRun.HeadSHA), 8)]
					ts := time.Unix(activeRun.CreatedAt, 0).Format(time.DateTime)
					fmt.Fprintf(w, "  %s  %s\n", sDim.Render("     id:"), activeRun.ID)
					fmt.Fprintf(w, "  %s  %s\n", sDim.Render(" branch:"), activeRun.Branch)
					fmt.Fprintf(w, "  %s  %s\n", sDim.Render(" status:"), runStatusStyle(activeRun.Status))
					steps, err := d.GetStepsByRun(activeRun.ID)
					if err != nil {
						return "", fmt.Errorf("load active run steps: %w", err)
					}
					rv := runViewFromDB(activeRun, steps)
					rv.GateAutomation = gateAutomationFromDB(d, activeRun, rv, types.ApprovalSurfaceTerminal)
					boundary := rv.Boundary
					fmt.Fprintf(w, "  %s  %s (%s)\n", sDim.Render("boundary:"), boundary.Status, boundary.Reason)
					printStatusAutomation(w, rv.GateAutomation)
					fmt.Fprintf(w, "  %s  %s\n", sDim.Render("   head:"), sDim.Render(sha))
					fmt.Fprintf(w, "  %s  %s\n", sDim.Render("started:"), sDim.Render(ts))
				} else {
					fmt.Fprintf(w, "\n  %s\n", sDim.Render("no active run"))
				}

				return "success", nil
			})
		},
	}
}

func printStatusAutomation(w io.Writer, automation *types.GateAutomation) {
	if automation == nil {
		return
	}
	fmt.Fprintf(w, "  %s  %s\n", sDim.Render("automation:"), automation.Status)
	fmt.Fprintf(w, "  %s  %s\n", sDim.Render("      mode:"), automation.RequestedMode)
	if automation.GateID != "" {
		fmt.Fprintf(w, "  %s  %s\n", sDim.Render("      gate:"), automation.GateID)
	}
	if automation.Reason != "" {
		fmt.Fprintf(w, "  %s  %s\n", sDim.Render("    reason:"), automation.Reason)
	}
	if automation.Message != "" {
		fmt.Fprintf(w, "  %s  %s\n", sDim.Render("   message:"), automation.Message)
	}
	for _, option := range automation.RecoveryOptions {
		fmt.Fprintf(w, "  %s  %s\n", sDim.Render("      help:"), option)
	}
}

func minLen(a, b int) int {
	if a < b {
		return a
	}
	return b
}

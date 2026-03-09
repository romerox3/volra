package main

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/romerox3/volra/internal/audit"
	"github.com/romerox3/volra/internal/output"
	"github.com/spf13/cobra"
)

var (
	auditAction string
	auditAgent  string
	auditSince  string
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "View deployment audit log",
	Long:  "Display the append-only audit log of deploy, down, and gateway actions.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		p := newPresenter()
		defer flushPresenter(p)
		return runAudit(p)
	},
}

func init() {
	auditCmd.Flags().StringVar(&auditAction, "action", "", "Filter by action (deploy, down, gateway)")
	auditCmd.Flags().StringVar(&auditAgent, "agent", "", "Filter by agent name")
	auditCmd.Flags().StringVar(&auditSince, "since", "", "Filter entries after date (YYYY-MM-DD)")
	rootCmd.AddCommand(auditCmd)
}

func runAudit(p output.Presenter) error {
	filter := &audit.Filter{
		Action: auditAction,
		Agent:  auditAgent,
	}
	if auditSince != "" {
		t, err := time.Parse("2006-01-02", auditSince)
		if err != nil {
			return fmt.Errorf("invalid --since date (use YYYY-MM-DD): %w", err)
		}
		filter.Since = t
	}

	entries, err := audit.Read(".", filter)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		p.Result("No audit entries found")
		return nil
	}

	if jsonOutput {
		data, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return fmt.Errorf("encoding audit entries: %w", err)
		}
		p.Result(string(data))
		return nil
	}

	// Human-readable table.
	var buf []byte
	w := tabwriter.NewWriter(writerFunc(func(b []byte) (int, error) {
		buf = append(buf, b...)
		return len(b), nil
	}), 0, 2, 2, ' ', 0)

	fmt.Fprintln(w, "Time\tAction\tAgent\tResult\tDuration")
	for _, e := range entries {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%dms\n",
			e.Timestamp.Format("2006-01-02 15:04:05"),
			e.Action, e.Agent, e.Result, e.DurationMs)
	}
	w.Flush()

	p.Result(string(buf))
	return nil
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(b []byte) (int, error) { return f(b) }

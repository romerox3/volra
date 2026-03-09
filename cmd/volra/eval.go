package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/romerox3/volra/internal/eval"
	"github.com/spf13/cobra"
)

var evalCmd = &cobra.Command{
	Use:   "eval [record|run] [path]",
	Short: "Evaluate agent performance against baselines",
	Long:  "Record metric baselines from Prometheus and detect regressions by comparing current values against recorded baselines.",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		subcmd := args[0]
		if subcmd != "record" && subcmd != "run" {
			return fmt.Errorf("unknown subcommand %q — use 'record' or 'run'", subcmd)
		}

		dir := "."
		if len(args) > 1 {
			dir = args[1]
		}

		p := newPresenter()
		defer flushPresenter(p)

		promURL := "http://localhost:9090"
		querier := &eval.PromClient{
			BaseURL: promURL,
			Client:  &http.Client{Timeout: 10 * time.Second},
		}

		return eval.Run(cmd.Context(), dir, subcmd, p, querier, jsonOutput)
	},
}

func init() {
	rootCmd.AddCommand(evalCmd)
}

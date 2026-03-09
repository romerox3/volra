package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/romerox3/volra/internal/controlplane"
	"github.com/romerox3/volra/internal/output"
	"github.com/spf13/cobra"
)

var serverPort int

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the Volra control plane server",
	Long:  "Start the REST API server that aggregates agent state, metrics, and deployment actions.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		p := newPresenter()
		defer flushPresenter(p)
		return runServer(p)
	},
}

func init() {
	serverCmd.Flags().IntVar(&serverPort, "port", 4441, "Port for the control plane API")
	rootCmd.AddCommand(serverCmd)
}

func runServer(p output.Presenter) error {
	dbPath := controlplane.DefaultDBPath()
	store, err := controlplane.NewStore(dbPath)
	if err != nil {
		return &output.UserError{
			Code: output.CodeControlPlaneDBFailed,
			What: fmt.Sprintf("Failed to initialize database: %v", err),
			Fix:  fmt.Sprintf("Check permissions on %s", filepath.Dir(dbPath)),
		}
	}
	defer store.Close()

	// Import legacy registry if it exists.
	home, _ := os.UserHomeDir()
	legacyPath := filepath.Join(home, ".volra", "agents.json")
	if n, err := store.ImportFromLegacyRegistry(legacyPath); err != nil {
		p.Warn(&output.UserWarning{
			What:    fmt.Sprintf("Failed to import legacy registry: %v", err),
			Assumed: "Continuing without legacy agents",
		})
	} else if n > 0 {
		p.Progress(fmt.Sprintf("Imported %d agents from legacy registry", n))
	}

	srv := controlplane.NewServer(store, serverPort)

	// Graceful shutdown on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	p.Result(fmt.Sprintf("Control plane running on http://localhost:%d", serverPort))

	select {
	case <-ctx.Done():
		p.Progress("Shutting down...")
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		p.Result("Server stopped")
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return &output.UserError{
			Code: output.CodeControlPlaneStartFailed,
			What: fmt.Sprintf("Server failed: %v", err),
			Fix:  fmt.Sprintf("Check if port %d is in use", serverPort),
		}
	}
}

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/romerox3/volra/internal/gateway"
	"github.com/romerox3/volra/internal/output"
	"github.com/romerox3/volra/internal/registry"
	"github.com/spf13/cobra"
)

var gatewayPort int

var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "Manage MCP gateway for multi-agent tool routing",
	Long:  "Start, stop, or inspect the MCP gateway that routes tool calls across registered agents.",
}

var gatewayStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start MCP gateway server",
	RunE: func(cmd *cobra.Command, _ []string) error {
		p := newPresenter()
		defer flushPresenter(p)
		return runGatewayStart(cmd.Context(), p)
	},
}

var gatewayToolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List all tools in the gateway catalog",
	RunE: func(cmd *cobra.Command, _ []string) error {
		p := newPresenter()
		defer flushPresenter(p)
		return runGatewayTools(cmd.Context(), p)
	},
}

var gatewayReloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Rebuild tool catalog from registered agents",
	RunE: func(cmd *cobra.Command, _ []string) error {
		p := newPresenter()
		defer flushPresenter(p)
		return runGatewayReload(cmd.Context(), p)
	},
}

func init() {
	gatewayStartCmd.Flags().IntVarP(&gatewayPort, "port", "p", 4440, "Port to listen on")
	gatewayCmd.AddCommand(gatewayStartCmd)
	gatewayCmd.AddCommand(gatewayToolsCmd)
	gatewayCmd.AddCommand(gatewayReloadCmd)
	rootCmd.AddCommand(gatewayCmd)
}

func runGatewayStart(ctx context.Context, p output.Presenter) error {
	agents, err := registry.List()
	if err != nil {
		return fmt.Errorf("reading agent registry: %w", err)
	}

	spawner := &gateway.SubprocessSpawner{}
	cat, err := gateway.BuildCatalog(ctx, agents, spawner, p, gateway.DefaultDiscoveryTimeout)
	if err != nil {
		return err
	}

	dirs := make(map[string]string, len(agents))
	for _, a := range agents {
		dirs[a.Name] = a.ProjectDir
	}

	router := gateway.NewRouter(cat, &gateway.SubprocessBackend{}, dirs)
	srv := gateway.NewServer(router, version)

	addr := fmt.Sprintf(":%d", gatewayPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", addr, err)
	}

	tools := router.ListTools()
	p.Result(fmt.Sprintf("MCP Gateway listening on http://localhost:%d/mcp (%d tools from %d agents)", gatewayPort, len(tools), len(agents)))

	httpSrv := &http.Server{Handler: srv}

	// Graceful shutdown on SIGINT/SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		p.Progress("Shutting down gateway...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	}()

	if err := httpSrv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("gateway server error: %w", err)
	}
	p.Result("Gateway stopped")
	return nil
}

func runGatewayTools(ctx context.Context, p output.Presenter) error {
	agents, err := registry.List()
	if err != nil {
		return fmt.Errorf("reading agent registry: %w", err)
	}

	spawner := &gateway.SubprocessSpawner{}
	cat, err := gateway.BuildCatalog(ctx, agents, spawner, p, gateway.DefaultDiscoveryTimeout)
	if err != nil {
		return err
	}

	tools := cat.Tools()
	if jsonOutput {
		data, err := json.MarshalIndent(tools, "", "  ")
		if err != nil {
			return fmt.Errorf("encoding tools: %w", err)
		}
		p.Result(string(data))
	} else {
		for _, t := range tools {
			p.Result(fmt.Sprintf("  %s  %s", t.Tool.Name, t.Tool.Description))
		}
		p.Result(fmt.Sprintf("\n%d tools from %d agents", len(tools), len(agents)))
	}
	return nil
}

func runGatewayReload(ctx context.Context, p output.Presenter) error {
	agents, err := registry.List()
	if err != nil {
		return fmt.Errorf("reading agent registry: %w", err)
	}

	spawner := &gateway.SubprocessSpawner{}
	cat, err := gateway.BuildCatalog(ctx, agents, spawner, p, gateway.DefaultDiscoveryTimeout)
	if err != nil {
		return err
	}

	tools := cat.Tools()
	p.Result(fmt.Sprintf("Catalog rebuilt: %d tools from %d agents", len(tools), len(agents)))
	return nil
}

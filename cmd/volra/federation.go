package main

import (
	"context"
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/romerox3/volra/internal/controlplane"
	"github.com/romerox3/volra/internal/output"
	"github.com/spf13/cobra"
)

var federationAPIKey string

var federationCmd = &cobra.Command{
	Use:   "federation",
	Short: "Manage federated control plane peers",
	Long:  "Add, remove, and list remote Volra control planes for cross-server agent visibility.",
}

var federationAddCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Add a federation peer",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p := newPresenter()
		defer flushPresenter(p)
		return runFederationAdd(p, args[0])
	},
}

var federationRemoveCmd = &cobra.Command{
	Use:   "remove <url>",
	Short: "Remove a federation peer",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p := newPresenter()
		defer flushPresenter(p)
		return runFederationRemove(p, args[0])
	},
}

var federationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List federation peers",
	RunE: func(cmd *cobra.Command, _ []string) error {
		p := newPresenter()
		defer flushPresenter(p)
		return runFederationList(p)
	},
}

func init() {
	federationAddCmd.Flags().StringVar(&federationAPIKey, "key", "", "API key for authenticating with the remote peer")

	federationCmd.AddCommand(federationAddCmd)
	federationCmd.AddCommand(federationRemoveCmd)
	federationCmd.AddCommand(federationListCmd)
	rootCmd.AddCommand(federationCmd)
}

func runFederationAdd(p output.Presenter, peerURL string) error {
	// Health check first.
	client := controlplane.NewFederationClient()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p.Progress(fmt.Sprintf("Checking peer health at %s...", peerURL))
	if err := client.CheckPeerHealth(ctx, peerURL); err != nil {
		return &output.UserError{
			Code: output.CodeFederationPeerUnreachable,
			What: fmt.Sprintf("Peer unreachable: %v", err),
			Fix:  "Verify the URL is correct and the remote Volra server is running",
		}
	}

	store, err := openStore()
	if err != nil {
		return err
	}
	defer store.Close()

	peer := controlplane.FederationPeer{
		URL:     peerURL,
		Name:    peerURL,
		APIKey:  federationAPIKey,
		AddedAt: time.Now().UTC(),
	}
	if err := store.InsertPeer(peer); err != nil {
		return fmt.Errorf("adding peer: %w", err)
	}

	p.Result(fmt.Sprintf("Federation peer added: %s", peerURL))
	return nil
}

func runFederationRemove(p output.Presenter, peerURL string) error {
	store, err := openStore()
	if err != nil {
		return err
	}
	defer store.Close()

	if err := store.DeletePeer(peerURL); err != nil {
		return err
	}

	p.Result(fmt.Sprintf("Federation peer removed: %s", peerURL))
	return nil
}

func runFederationList(p output.Presenter) error {
	store, err := openStore()
	if err != nil {
		return err
	}
	defer store.Close()

	peers, err := store.ListPeers()
	if err != nil {
		return err
	}

	if len(peers) == 0 {
		p.Result("No federation peers configured")
		return nil
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(peers, "", "  ")
		p.Result(string(data))
		return nil
	}

	var buf []byte
	w := tabwriter.NewWriter(writerFunc(func(b []byte) (int, error) {
		buf = append(buf, b...)
		return len(b), nil
	}), 0, 2, 2, ' ', 0)

	fmt.Fprintln(w, "URL\tName\tAdded")
	for _, peer := range peers {
		fmt.Fprintf(w, "%s\t%s\t%s\n", peer.URL, peer.Name, peer.AddedAt.Format("2006-01-02"))
	}
	w.Flush()

	p.Result(string(buf))
	return nil
}

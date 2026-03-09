package main

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/romerox3/volra/internal/auth"
	"github.com/romerox3/volra/internal/controlplane"
	"github.com/romerox3/volra/internal/output"
	"github.com/spf13/cobra"
)

var (
	authKeyRole string
	authKeyName string
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage API key authentication",
	Long:  "Create, list, and revoke API keys for the control plane.",
}

var authCreateKeyCmd = &cobra.Command{
	Use:   "create-key",
	Short: "Create a new API key",
	RunE: func(cmd *cobra.Command, _ []string) error {
		p := newPresenter()
		defer flushPresenter(p)
		return runAuthCreateKey(p)
	},
}

var authListKeysCmd = &cobra.Command{
	Use:   "list-keys",
	Short: "List all API keys",
	RunE: func(cmd *cobra.Command, _ []string) error {
		p := newPresenter()
		defer flushPresenter(p)
		return runAuthListKeys(p)
	},
}

var authRevokeKeyCmd = &cobra.Command{
	Use:   "revoke-key <id>",
	Short: "Revoke an API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p := newPresenter()
		defer flushPresenter(p)
		return runAuthRevokeKey(p, args[0])
	},
}

func init() {
	authCreateKeyCmd.Flags().StringVar(&authKeyRole, "role", "viewer", "Key role: admin, operator, viewer")
	authCreateKeyCmd.Flags().StringVar(&authKeyName, "name", "", "Key name (required)")
	_ = authCreateKeyCmd.MarkFlagRequired("name")

	authCmd.AddCommand(authCreateKeyCmd)
	authCmd.AddCommand(authListKeysCmd)
	authCmd.AddCommand(authRevokeKeyCmd)
	rootCmd.AddCommand(authCmd)
}

func openStore() (*controlplane.Store, error) {
	return controlplane.NewStore(controlplane.DefaultDBPath())
}

func runAuthCreateKey(p output.Presenter) error {
	role := auth.Role(authKeyRole)
	if !auth.ValidRoles[role] {
		return fmt.Errorf("invalid role %q: must be admin, operator, or viewer", authKeyRole)
	}

	id, plaintext, hash, err := auth.GenerateKey()
	if err != nil {
		return fmt.Errorf("generating key: %w", err)
	}

	store, err := openStore()
	if err != nil {
		return err
	}
	defer store.Close()

	key := controlplane.APIKey{
		ID:        id,
		Name:      authKeyName,
		KeyHash:   hash,
		Role:      string(role),
		CreatedAt: time.Now().UTC(),
	}
	if err := store.InsertAPIKey(key); err != nil {
		return fmt.Errorf("storing key: %w", err)
	}

	if jsonOutput {
		result := auth.KeyResult{
			ID:        id,
			Name:      authKeyName,
			Key:       plaintext,
			Role:      string(role),
			CreatedAt: key.CreatedAt,
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		p.Result(string(data))
	} else {
		p.Result(fmt.Sprintf("API key created:\n  ID:   %s\n  Name: %s\n  Role: %s\n  Key:  %s\n\nSave this key — it cannot be shown again.", id, authKeyName, role, plaintext))
	}
	return nil
}

func runAuthListKeys(p output.Presenter) error {
	store, err := openStore()
	if err != nil {
		return err
	}
	defer store.Close()

	keys, err := store.ListAPIKeys()
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		p.Result("No API keys configured")
		return nil
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(keys, "", "  ")
		p.Result(string(data))
		return nil
	}

	var buf []byte
	w := tabwriter.NewWriter(writerFunc(func(b []byte) (int, error) {
		buf = append(buf, b...)
		return len(b), nil
	}), 0, 2, 2, ' ', 0)

	fmt.Fprintln(w, "ID\tName\tRole\tCreated\tRevoked")
	for _, k := range keys {
		revoked := "-"
		if k.RevokedAt != nil {
			revoked = k.RevokedAt.Format("2006-01-02")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			k.ID, k.Name, k.Role,
			k.CreatedAt.Format("2006-01-02"), revoked)
	}
	w.Flush()

	p.Result(string(buf))
	return nil
}

func runAuthRevokeKey(p output.Presenter, id string) error {
	store, err := openStore()
	if err != nil {
		return err
	}
	defer store.Close()

	if err := store.RevokeAPIKey(id); err != nil {
		return err
	}

	p.Result(fmt.Sprintf("API key %s revoked", id))
	return nil
}

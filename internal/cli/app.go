// Package cli wires fortyhours' cobra commands to the config and Productive
// client packages.
package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/cbarber/fortyhours/internal/config"
	"github.com/cbarber/fortyhours/internal/productive"
	"github.com/spf13/cobra"
)

// App bundles the dependencies every subcommand needs: loaded config, an
// authenticated Productive client, and where to write output.
type App struct {
	Config *config.Config
	Client *productive.Client
	Out    io.Writer
	JSON   bool
}

// newApp loads config and builds a Productive client for cmd. Commands that
// don't need Productive access (init, when starting from scratch) should
// call config.Load directly instead.
func newApp(cmd *cobra.Command) (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if cfg.APIToken == "" || cfg.OrgID == "" {
		return nil, fmt.Errorf(
			"missing Productive credentials: set %s and %s, or run `fortyhours init`",
			config.EnvAPIToken, config.EnvOrgID,
		)
	}

	jsonOut, err := cmd.Flags().GetBool("json")
	if err != nil {
		jsonOut = false
	}

	client := newProductiveClient(cmd, cfg.APIToken, cfg.OrgID)

	return &App{
		Config: cfg,
		Client: client,
		Out:    cmd.OutOrStdout(),
		JSON:   jsonOut,
	}, nil
}

// newProductiveClient builds a Productive client honoring the --debug flag/
// FORTYHOURS_DEBUG env var and the FORTYHOURS_BASE_URL test override.
func newProductiveClient(cmd *cobra.Command, apiToken, orgID string) *productive.Client {
	client := productive.NewClient(apiToken, orgID)
	if baseURL := os.Getenv("FORTYHOURS_BASE_URL"); baseURL != "" {
		client.BaseURL = baseURL
	}
	debug, _ := cmd.Flags().GetBool("debug")
	if os.Getenv("FORTYHOURS_DEBUG") != "" {
		debug = true
	}
	client.Debug = debug
	client.DebugWriter = cmd.ErrOrStderr()
	return client
}

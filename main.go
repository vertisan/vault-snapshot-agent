package main

import (
	"context"
	"os"

	"charm.land/log/v2"
	"github.com/urfave/cli/v3"
	"github.com/vertisan/vault-snapshot-agent/internal/config"
	"github.com/vertisan/vault-snapshot-agent/pkg/agent"
)

var version = "dev"

func GetVersion() string {
	return version
}

func main() {
	app := &cli.Command{
		Name:        "vault-snapshot-agent",
		Description: "A custom Vault Agent for managing snapshots automatically.",
		Version:     GetVersion(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Load configuration from `FILE`",
				Value:   config.DefaultConfigPath,
				Sources: cli.EnvVars("VAULT_SNAPSHOT_AGENT_CONFIG"),
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			agent.Agent(cmd.String("config"))
			return nil
		},
	}

	err := app.Run(context.Background(), os.Args)
	if err != nil {
		log.Fatal("Cannot start Vault Agent Snapshot!", "err", err)
	}
}

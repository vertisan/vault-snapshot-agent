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

func newRootCommand() *cli.Command {
	return &cli.Command{
		Name:        "vault-snapshot-agent",
		Usage:       "Manage HashiCorp Vault Raft snapshots",
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
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "Take a Vault Raft snapshot and ship it to the configured storages",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					agent.Agent(cmd.String("config"))
					return nil
				},
			},
		},
		// Without an explicit Action the library falls back to showing help and exiting 0,
		// which would let a stale `ExecStart=` silently take no snapshots every hour.
		Action: func(ctx context.Context, cmd *cli.Command) error {
			_ = cli.ShowRootCommandHelp(cmd)
			return cli.Exit("", 1)
		},
	}
}

func main() {
	err := newRootCommand().Run(context.Background(), os.Args)
	if err != nil {
		log.Fatal("Cannot start Vault Agent Snapshot!", "err", err)
	}
}

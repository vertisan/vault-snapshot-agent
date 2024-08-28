package main

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/urfave/cli/v2"
	"github.com/vertisan/vault-snapshot-agent/internal/config"
	"github.com/vertisan/vault-snapshot-agent/pkg/agent"
)

func main() {
	app := cli.NewApp()
	app.Name = "vault-snapshot-agent"
	app.Description = "A custom Vault Agent for managing snapshots automatically."
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Load configuration from `FILE`",
			Value:   config.DefaultConfigPath,
			EnvVars: []string{"VAULT_SNAPSHOT_AGENT_CONFIG"},
		},
	}
	app.Action = func(cCtx *cli.Context) error {
		agent.Agent(cCtx.String("config"))

		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal("Cannot start Vault Agent Snapshot!", "err", err)
	}
}

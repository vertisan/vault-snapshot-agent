package main

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
	"github.com/vertisan/vault-snapshot-agent/internal/config"
)

func TestGetVersion(t *testing.T) {
	version := GetVersion()
	assert.NotEmpty(t, version)
	// Default version should be "dev" when not built with ldflags
	assert.Equal(t, "dev", version)
}

func TestNewRootCommand(t *testing.T) {
	t.Run("Reports the current version", func(t *testing.T) {
		assert.Equal(t, GetVersion(), newRootCommand().Version)
	})

	t.Run("Registers the run subcommand", func(t *testing.T) {
		cmd := newRootCommand().Command("run")

		assert.NotNil(t, cmd)
		assert.NotNil(t, cmd.Action)
	})

	t.Run("Exposes config flag to subcommands", func(t *testing.T) {
		flag := newRootCommand().Flags[0]

		// urfave/cli v3 inherits a parent flag by subcommands unless it is marked
		// local, so `run` reads --config without redeclaring it.
		local, ok := flag.(cli.LocalFlag)
		assert.True(t, ok)
		assert.False(t, local.IsLocal())
		assert.Contains(t, flag.Names(), "config")
	})

	t.Run("Defaults config flag to the packaged path", func(t *testing.T) {
		cmd := newRootCommand()

		assert.Equal(t, config.DefaultConfigPath, cmd.String("config"))
	})
}

func TestRootActionRequiresSubcommand(t *testing.T) {
	cmd := newRootCommand()
	// Run() would normally default this; the action is called directly here so
	// the library's exit handler cannot terminate the test binary.
	cmd.Writer = io.Discard

	err := cmd.Action(context.Background(), cmd)

	assert.Error(t, err)

	exitErr, ok := err.(cli.ExitCoder)
	assert.True(t, ok)
	assert.Equal(t, 1, exitErr.ExitCode())
	// An empty message keeps HandleExitCoder from printing after the help text.
	assert.Empty(t, err.Error())
}

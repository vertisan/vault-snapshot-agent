package snapshot

import (
	"github.com/vertisan/vault-snapshot-agent/internal/config"
	"github.com/vertisan/vault-snapshot-agent/internal/vault"
)

type Snapshot struct {
	Vault vault.Vault
}

func NewSnapshot(config *config.Configuration) (*Snapshot, error) {
	snapshotter := &Snapshot{}

	// Initialize Vault
	err := snapshotter.Vault.NewClient(&config.Vault)
	if err != nil {
		return nil, err
	}

	return snapshotter, nil
}

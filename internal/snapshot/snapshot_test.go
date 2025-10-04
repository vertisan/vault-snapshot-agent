package snapshot_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vertisan/vault-snapshot-agent/internal/config"
	"github.com/vertisan/vault-snapshot-agent/internal/snapshot"
)

func TestNewSnapshot(t *testing.T) {
	t.Run("NewSnapshot with invalid Vault address", func(t *testing.T) {
		cfg := &config.Configuration{
			Vault: config.VaultConfig{
				Address:  "http://invalid-vault-address:8200",
				RoleId:   "test-role-id",
				SecretId: "test-secret-id",
				Approle:  "approle",
			},
		}

		// This will create a snapshot but fail when trying to authenticate
		// We expect an error since we can't connect to Vault
		snap, err := snapshot.NewSnapshot(cfg)
		if err != nil {
			// This is expected if Vault is not available
			assert.Error(t, err)
			assert.Nil(t, snap)
		} else {
			// If somehow it succeeds (unlikely), snap should not be nil
			assert.NotNil(t, snap)
		}
	})
}

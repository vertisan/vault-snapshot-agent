package vault

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/log"
	vaultApi "github.com/hashicorp/vault/api"
	"github.com/vertisan/vault-snapshot-agent/internal/config"
)

type Vault struct {
	API             *vaultApi.Client
	TokenExpiration time.Time
}

func (v *Vault) NewClient(config *config.VaultConfig) error {
	vaultConfig := vaultApi.DefaultConfig()
	if config.Address != "" {
		vaultConfig.Address = config.Address
	}

	api, err := vaultApi.NewClient(vaultConfig)
	if err != nil {
		return err
	}

	v.API = api

	return v.SetClientToken(config)
}

func (v *Vault) SetClientToken(config *config.VaultConfig) error {
	approle := "approle"
	data := map[string]interface{}{
		"role_id":   config.RoleId,
		"secret_id": config.SecretId,
	}

	if config.Approle != "" {
		approle = config.Approle
	}

	resp, err := v.API.Logical().Write(fmt.Sprintf("auth/%s/login", approle), data)
	if err != nil {
		return fmt.Errorf("cannot login into Vault with AppRole: %v", err.Error())
	}

	v.API.SetToken(resp.Auth.ClientToken)
	v.TokenExpiration = time.Now().Add((time.Second * time.Duration(resp.Auth.LeaseDuration)) / 2)

	return nil
}

func (v *Vault) IsLeader() bool {
	leader, err := v.API.Sys().Leader()

	if err != nil {
		log.Error("Cannot determine leader instance! Vault Snapshotter will run only on the current leader node.", "err", err)
		os.Exit(1)
	}

	return leader.IsSelf
}

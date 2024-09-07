package config

import (
	"os"

	"github.com/charmbracelet/log"
	"gopkg.in/yaml.v3"
)

type Configuration struct {
	Vault   VaultConfig   `yaml:"vault"`
	Storage StorageConfig `yaml:"storage"`
}

type VaultConfig struct {
	Address  string `yaml:"addr" default:"https://127.0.0.1:8200"`
	RoleId   string `yaml:"roleId"`
	SecretId string `yaml:"secretId"`
	Approle  string `yaml:"approle" default:"approle"`
}

type StorageConfig struct {
	Retention int                `yaml:"retention,omitempty"`
	Local     LocalStorageConfig `yaml:"local,omitempty"`
}

type LocalStorageConfig struct {
	Path string `yaml:"path"`
}

const (
	DefaultConfigPath = "/etc/vault.d/vault-snapshot-agent.yaml"
)

func ReadConfig(configPath string) (*Configuration, error) {
	fileContent, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Cannot read configuration: %v", err)
	}

	config := &Configuration{}
	err = yaml.Unmarshal(fileContent, &config)
	if err != nil {
		log.Fatalf("Cannot parse configuration: %v", err)
	}

	log.Debug("Configuration has been loaded!")

	return config, nil
}

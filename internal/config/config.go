package config

import (
	"log"
	"os"

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
	Local LocalStorageConfig `yaml:"local"` // omitempty if another storage is enabled
}

type LocalStorageConfig struct {
	Path string `yaml:"path"`
}

func ReadConfig() (*Configuration, error) {
	// TODO: Allow to be passed from CLI
	// file := "/etc/vault.d/vault-snapshot-agent.yaml"
	file := "./.local-data/vault-snapshot-agent.yaml"

	fileContent, err := os.ReadFile(file)
	if err != nil {
		log.Fatalf("Cannot read configuration from file at '%s': %v", file, err.Error())
	}

	config := &Configuration{}
	err = yaml.Unmarshal(fileContent, &config)
	if err != nil {
		log.Fatalf("Cannot parse configuration: %v", err.Error())
	}

	log.Println("Configuration has been loaded!")

	return config, nil
}

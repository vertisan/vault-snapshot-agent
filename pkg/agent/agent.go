package agent

import (
	"bytes"

	"github.com/charmbracelet/log"

	"github.com/vertisan/vault-snapshot-agent/internal/config"
	"github.com/vertisan/vault-snapshot-agent/internal/logger"
	"github.com/vertisan/vault-snapshot-agent/internal/snapshot"
	"github.com/vertisan/vault-snapshot-agent/internal/storage"
)

func Agent(configPath string) {
	logger.NewLogger()

	log.Debug("Loading configuration ...")

	c, err := config.ReadConfig(configPath)
	if err != nil {
		log.Fatal("Cannot load configuration!", "err", err)
	}

	storageManager, err := storage.NewStorageManager(&c.Storage)
	if err != nil {
		log.Fatal("Cannot initialize Storage Manager!", "err", err)
	}

	log.Info("Snapshot agent running")
	snapshotAgent, err := snapshot.NewSnapshot(c)
	if err != nil {
		log.Fatal("Cannot initialize Snapshot!", "err", err)
	}

	log.Info("Waiting to obtain leadership...")
	if snapshotAgent.IsOnLeader() {
		log.Info("Obtained leadership")

		var raftData bytes.Buffer
		err := snapshotAgent.SnapRaft(&raftData)
		if err != nil {
			log.Fatal("Unable to generate snapshot!", "err", err)
		}

		fileName := storageManager.SaveFile(raftData.Bytes())
		log.Info("Saved snapshot", "fileName", fileName)

		if c.Storage.Retention > 0 {
			log.Info("Removing old snapshots...")

			if err := storageManager.Cleanup(c.Storage.Retention); err != nil {
				log.Error("Cannot clean up storage(s) from older snapshots!", "err", err.Error())
			}
		}
	} else {
		log.Info("Snapshot agent is not running on leader node, skipping ...")
	}
}

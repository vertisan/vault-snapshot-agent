package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/vertisan/vault-snapshot-agent/internal/config"
	"github.com/vertisan/vault-snapshot-agent/pkg/snapshot"
)

// Handle version

func main() {
	log.Println("Loading configuration ...")

	c, err := config.ReadConfig()
	if err != nil {
		log.Fatalln("Cannot load configuration!")
	}

	log.Println("Preparing snapshot agent ...")
	snapshotAgent, err := snapshot.NewSnapshot(c)
	if err != nil {
		log.Fatalf("Cannot start Vault Snapshotter! %v", err.Error())
	}

	log.Println("Checking if the selected Vault is a leader ...")
	if snapshotAgent.Vault.IsLeader() {
		log.Println("Leader detected! Creating snapshot ...")

		var raftData bytes.Buffer
		err := snapshotAgent.Vault.API.Sys().RaftSnapshot(&raftData)
		if err != nil {
			log.Fatalln("Unable to generate snapshot", err.Error())
		}

		// PoC
		t := time.Now()
		fileName := fmt.Sprintf("./vault-snapshot.%s.snap", t.Format("200601021504"))
		err = os.WriteFile(fileName, raftData.Bytes(), 0644)
		if err != nil {
			log.Fatalln("Cannot save snapshot!")
		}
		// end PoC

		log.Println("Snapshot has been made!")
	} else {
		log.Println("Vault Snapshotter is not running on leader node, skipping ...")
	}
}

package logger

import "github.com/charmbracelet/log"

func NewLogger() {
	// TODO: Allow debug to be enabled from CLI
	log.SetLevel(log.DebugLevel)
	log.SetTimeFormat("2006-01-02 15:04:05")
}

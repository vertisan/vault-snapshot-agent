package storage

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
)

type LocalStorageDriver struct {
	Path string
}

func (l *LocalStorageDriver) SaveFile(fileName string, data []byte) (string, error) {
	fullPath := fmt.Sprintf("%s/%s", l.Path, fileName)

	err := os.WriteFile(fullPath, data, 0400)
	if err != nil {
		log.Error("Cannot save file!", "err", err.Error())
		return "", err
	}

	return fullPath, nil
}

package storage

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/log"
)

type LocalStorageDriver struct {
	Path string
}

func (l *LocalStorageDriver) Name() string {
	return "local"
}

func (l *LocalStorageDriver) Write(fileName string, data []byte) (string, error) {
	fullPath := fmt.Sprintf("%s/%s", l.Path, fileName)

	if err := os.WriteFile(fullPath, data, 0400); err != nil {
		log.Fatal("Cannot save file!", "err", err.Error())
	}

	return fullPath, nil
}

func (l *LocalStorageDriver) Remove(fileName string) error {
	fullPath := fmt.Sprintf("%s/%s", l.Path, fileName)

	if err := os.Remove(fullPath); err != nil {
		log.Error("Cannot remove file!", "err", err.Error())
		return err
	}

	return nil
}

func (l *LocalStorageDriver) List() ([]FileInfo, error) {
	dirEntries, err := os.ReadDir(l.Path)
	if err != nil {
		log.Error("Cannot get items from directory!", "err", err.Error())
		return nil, err
	}

	files := make([]FileInfo, 0)
	for _, file := range dirEntries {
		if !file.IsDir() && strings.HasPrefix(file.Name(), "vault-snapshot-") && strings.HasSuffix(file.Name(), ".snap") {
			fileDetails, _ := file.Info()

			files = append(files, FileInfo{
				Name:    file.Name(),
				ModTime: fileDetails.ModTime(),
				Size:    fileDetails.Size(),
			})
		}
	}

	return files, nil
}

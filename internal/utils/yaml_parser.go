package utils

import (
	"bytes"
	"errors"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

func ParseYamlData[T any](data []byte) (*T, error) {
	config := new(T)

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(config); err != nil {
		if err == io.EOF {
			return config, nil
		}

		// Remove new lines from the error log
		return nil, errors.New(strings.ReplaceAll(err.Error(), "\n", ""))
	}

	return config, nil
}

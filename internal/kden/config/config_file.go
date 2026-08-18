package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

const configFileName = "config.json"

type fileFunctions struct {
	search func(string) (string, error)
	create func(string) (string, error)
	write  func(string, []byte, os.FileMode) error
}

var fileFuncs = fileFunctions{
	search: xdg.SearchConfigFile,
	create: xdg.ConfigFile,
	write:  os.WriteFile,
}

func getOrCreateConfigFile() (string, error) {
	configFilePath, err := fileFuncs.search(filepath.Join(RootCommandName, configFileName))
	if err != nil {
		filePath, err := createConfigFile()
		if err != nil {
			return "", err
		}
		return filePath, nil

	}
	return configFilePath, nil
}

func createConfigFile() (string, error) {
	configFilePath, err := fileFuncs.create(filepath.Join(RootCommandName, configFileName))
	if err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	err = fileFuncs.write(configFilePath, []byte("{}"), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create config file: %w", err)
	}
	return configFilePath, nil
}

func updateConfigFile(configFilePath string, data []byte) error {
	err := fileFuncs.write(configFilePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

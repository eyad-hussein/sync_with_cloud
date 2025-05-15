package internal

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	CredentialsFile string            `yaml:"credentials_file"`
	RootFolderId    string            `yaml:"root_folder_id"`
	Paths           map[string]string `yaml:"paths"`
	Exclude         map[string]string `yaml:"exclude"`
}

func (c *Config) ValidateConfig() error {
	if _, err := os.Stat(c.CredentialsFile); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("credentials file %s does not exist", c.CredentialsFile)
	}

	if c.RootFolderId == "" {
		return fmt.Errorf("root_folder_id cannot be empty")
	}

	if len(c.Paths) == 0 {
		return fmt.Errorf("no paths specified for syncing")
	}

	for localPath := range c.Paths {
		if _, err := os.Stat(localPath); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("folder/file %s does not exist locally", localPath)
		}
	}

	for excludePath := range c.Exclude {
		isUnderIncludedPath := false
		for includedPath := range c.Paths {
			if strings.HasPrefix(excludePath, includedPath) {
				isUnderIncludedPath = true
				break
			}
		}

		if !isUnderIncludedPath {
			return fmt.Errorf("excluded path %s is not under any included path", excludePath)
		}
	}

	return nil
}

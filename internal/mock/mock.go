package mock

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Mock struct {
	Paths   []string          `json:"paths"`
	Verb    string            `json:"verb"`
	Body    any               `json:"body"`
	Headers map[string]string `json:"headers"`
	Status  int               `json:"status"`
}

func LoadMocksFromFS(directory string) ([]Mock, error) {
	files, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	mocks := []Mock{}
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			filePath := filepath.Join(directory, file.Name())
			fmt.Printf("Reading file: %s\n", filePath)

			content, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Printf("Error reading file %s: %s\n", filePath, err)
				continue
			}

			var mock Mock
			if err := json.Unmarshal(content, &mock); err != nil {
				fmt.Printf("Error unmarshalling JSON in file %s: %s\n", filePath, err)
				continue
			}
			mocks = append(mocks, mock)
		}
	}
	return mocks, nil
}

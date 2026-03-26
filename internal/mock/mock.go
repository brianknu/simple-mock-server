package mock

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Mock struct {
	Paths            []string          `json:"paths"`
	Verb             string            `json:"verb"`
	Body             any               `json:"body"`
	Headers          map[string]string `json:"headers"`
	Status           int               `json:"status"`
	PrintRequestBody bool              `json:"print_request_body"`
	ResponseTime     int               `json:"response_time"`
	SourceFile       string            `json:"-"`
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
			log.Printf("Using %s\n", filePath)

			content, err := os.ReadFile(filePath)
			if err != nil {
				log.Printf("Error reading file %s: %s\n", filePath, err)
				continue
			}

			var mock Mock
			if err := json.Unmarshal(content, &mock); err != nil {
				log.Printf("Error unmarshalling JSON in file %s: %s\n", filePath, err)
				continue
			}
			mock.SourceFile = filePath
			mocks = append(mocks, mock)
		}
	}
	return mocks, nil
}

// SaveMockToFS writes a mock as indented JSON to the given directory.
// If the mock has a SourceFile set, it overwrites that file.
// Otherwise, it generates a filename from the verb and first path.
func SaveMockToFS(directory string, m Mock) (string, error) {
	var filename string
	if m.SourceFile != "" {
		filename = m.SourceFile
	} else {
		name := "mock"
		if len(m.Paths) > 0 {
			name = strings.ReplaceAll(strings.Trim(m.Paths[0], "/"), "/", "_")
		}
		base := filepath.Join(directory, fmt.Sprintf("%s_%s", m.Verb, name))
		filename = base + ".json"
		for i := 2; ; i++ {
			if _, err := os.Stat(filename); os.IsNotExist(err) {
				break
			}
			filename = fmt.Sprintf("%s_%d.json", base, i)
		}
	}

	data, err := json.MarshalIndent(m, "", "    ")
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return "", err
	}
	return filename, nil
}

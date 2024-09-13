package utils

import (
	"fmt"
	"os"
)

func ResolveDirectoryWithMocks(dirFromCMD string) (string, error) {
	if dirFromCMD == "" {
		dirFromEnv := os.Getenv("SIMPLE_MOCKS_LOCATION")
		if dirFromEnv == "" {
			return "", fmt.Errorf("directory not specified, use -d flag or set SIMPLE_MOCKS_LOCATION environment variable")
		} else {
			return dirFromEnv, nil
		}
	} else {
		return dirFromCMD, nil
	}
}
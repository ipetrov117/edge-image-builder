package podman

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

func generatePodmanLogFile(fileName, out string) (*os.File, error) {
	timestamp := time.Now().Format("Jan02_15-04-05")
	filename := fmt.Sprintf(fileName, timestamp)
	logFilename := filepath.Join(out, filename)

	logFile, err := os.Create(logFilename)
	if err != nil {
		return nil, fmt.Errorf("creating podman log file: %w", err)
	}
	zap.L().Sugar().Debugf("podman log file created: %s", logFilename)

	return logFile, err
}

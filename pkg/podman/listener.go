package podman

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	podmanArgsBase        = "--log-level=debug system service -t 0"
	podmanExec            = "/usr/bin/podman"
	podmanListenerLogFile = "podman-system-service-%s.log"
)

func setupAPIListener(out string) error {
	cmd, logfile, err := preparePodmanCommand(out)
	if err != nil {
		return fmt.Errorf("configuring the podman system serice command: %w", err)
	}
	defer logfile.Close()

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("error running podman system service: %w", err)
	}

	return err
}

func preparePodmanCommand(out string) (*exec.Cmd, *os.File, error) {
	logFile, err := generatePodmanLogFile(podmanListenerLogFile, out)
	if err != nil {
		return nil, nil, fmt.Errorf("generating podman lister log file: %w", err)
	}

	args := strings.Split(podmanArgsBase, " ")
	cmd := exec.Command(podmanExec, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	return cmd, logFile, nil
}

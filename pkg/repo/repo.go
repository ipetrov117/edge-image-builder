package repo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/podman"
	"go.uber.org/zap"
)

const (
	createRepoExec = "/usr/bin/createrepo"
	createRepoLog  = "createrepo-%s.log"
)

func Create(ctx *image.Context) (string, error) {
	p, err := podman.New(ctx.BuildDir)
	if err != nil {
		return "", fmt.Errorf("starting podman client: %w", err)
	}

	resolverDir := filepath.Join(ctx.BuildDir, "resolver")
	if err := os.MkdirAll(resolverDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("creating %s dir: %w", resolverDir, err)
	}

	pkgResolver := &PkgResolver{
		Context:      resolverDir,
		BaseISO:      filepath.Join(ctx.ImageConfigDir, "images", ctx.ImageDefinition.Image.BaseImage),
		PkgToInstall: &ctx.ImageDefinition.OperatingSystem.Packages,
		RpmDir:       filepath.Join(ctx.ImageConfigDir, "rpms"),
		Podman:       p,
	}

	repoName, pkgList, err := pkgResolver.Resolve(ctx.CombustionDir)
	if err != nil {
		return "", fmt.Errorf("resolving packages: %w", err)
	}

	repoPath := filepath.Join(ctx.CombustionDir, repoName)

	if err := producePkgListFile(filepath.Join(repoPath, "package-list.txt"), pkgList); err != nil {
		return "", fmt.Errorf("producing package list file: %w", err)
	}

	if err := createPkgRepo(repoPath, ctx.BuildDir); err != nil {
		return "", fmt.Errorf("creating package repo: %w", err)
	}

	return repoPath, nil
}

func producePkgListFile(filePath string, pkgList []string) error {
	refactoredList := []string{}
	for _, pkg := range pkgList {
		if strings.HasSuffix(pkg, ".rpm") {
			pkg = strings.TrimSuffix(filepath.Base(pkg), filepath.Ext(pkg))
		}
		refactoredList = append(refactoredList, pkg)
	}

	list, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("creating package list file: %w", err)
	}

	defer list.Close()

	_, err = list.WriteString(strings.Join(refactoredList, " "))
	if err != nil {
		return fmt.Errorf("writing package list file: %w", err)
	}

	return nil
}

func createPkgRepo(path, logOut string) error {
	cmd, logfile, err := prepareRepoCommand(path, logOut)
	if err != nil {
		return fmt.Errorf("preparing createrepo command: %w", err)
	}
	defer logfile.Close()

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error running createrepo: %w", err)
	}

	return err
}

func prepareRepoCommand(path, logOut string) (*exec.Cmd, *os.File, error) {
	logFile, err := generateCreateRepoLog(logOut)
	if err != nil {
		return nil, nil, fmt.Errorf("generating createrepo log file: %w", err)
	}

	cmd := exec.Command(createRepoExec, path)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	return cmd, logFile, nil
}

func generateCreateRepoLog(out string) (*os.File, error) {
	timestamp := time.Now().Format("Jan02_15-04-05")
	filename := fmt.Sprintf(createRepoLog, timestamp)
	logFilename := filepath.Join(out, filename)

	logFile, err := os.Create(logFilename)
	if err != nil {
		return nil, fmt.Errorf("creating log file: %w", err)
	}
	zap.L().Sugar().Debugf("log file created: %s", logFilename)

	return logFile, err
}

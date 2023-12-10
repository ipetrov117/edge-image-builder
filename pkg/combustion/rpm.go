package combustion

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/suse-edge/edge-image-builder/pkg/resolver"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
)

const (
	userRPMsDir         = "rpms"
	modifyRPMScriptName = "10-rpm-install.sh"
	rpmComponentName    = "RPM"
	combustionBasePath  = "/dev/shm/combustion/config"
	createRepoExec      = "/usr/bin/createrepo"
	createRepoLog       = "createrepo-%s.log"
)

//go:embed templates/10-rpm-install.sh.tpl
var modifyRPMScript string

func configureRPMs(ctx *image.Context) ([]string, error) {
	if !isComponentConfigured(ctx, userRPMsDir) {
		log.AuditComponentSkipped(rpmComponentName)
		return nil, nil
	}

	// rpmSourceDir := generateComponentPath(ctx, userRPMsDir)

	res, err := resolver.New(ctx.BuildDir, filepath.Join(ctx.ImageConfigDir, "images", ctx.ImageDefinition.Image.BaseImage), "", &ctx.ImageDefinition.OperatingSystem.Packages)
	if err != nil {
		return nil, fmt.Errorf("preparing resolver: %w", err)
	}

	repoName, pkgList, err := res.Resolve(ctx.CombustionDir)
	if err != nil {
		return nil, fmt.Errorf("resolving package dependencies: %w", err)
	}

	repoPath := filepath.Join(ctx.CombustionDir, repoName)
	if err := createRPMRepo(repoPath, ctx.BuildDir); err != nil {
		return nil, fmt.Errorf("creating rpm repo from %s: %w", repoPath, err)
	}

	script, err := writeRPMScript(ctx, repoName, pkgList)
	if err != nil {
		log.AuditComponentFailed(rpmComponentName)
		return nil, fmt.Errorf("writing the RPM install script %s: %w", modifyRPMScriptName, err)
	}

	log.AuditComponentSuccessful(rpmComponentName)
	return []string{script}, nil
}

func writeRPMScript(ctx *image.Context, repoName string, pkgList []string) (string, error) {
	values := struct {
		RepoPath string
		RepoName string
		PKGList  string
	}{
		RepoPath: filepath.Join(combustionBasePath, repoName),
		RepoName: repoName,
		PKGList:  strings.Join(pkgList, " "),
	}

	data, err := template.Parse(modifyRPMScriptName, modifyRPMScript, &values)
	if err != nil {
		return "", fmt.Errorf("parsing RPM script template: %w", err)
	}

	filename := filepath.Join(ctx.CombustionDir, modifyRPMScriptName)
	err = os.WriteFile(filename, []byte(data), fileio.ExecutablePerms)
	if err != nil {
		return "", fmt.Errorf("writing RPM script: %w", err)
	}

	return modifyRPMScriptName, nil
}

func createRPMRepo(path, logOut string) error {
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

func needsResolution(pkg *image.Packages, rpmDir string) (bool, error) {
	rpmDirExists, err := dirExists(rpmDir)
	if err != nil {
		return false, fmt.Errorf("checking if rpm dir exists: %w", err)
	}

	// confiugred package list with either a third repo or registration code is provided
	if len(pkg.PKGList) > 0 && (len(pkg.AddRepos) > 0 || pkg.RegCode != "") {
		return true, nil
	}

	// rpm dir exists and either a third party repo or a registrtation code is provided
	if rpmDirExists && (len(pkg.AddRepos) > 0 || pkg.RegCode != "") {
		return true, nil
	}

	return false, nil

}

func dirExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("describing file at %s: %w", path, err)
	}
	return true, nil
}

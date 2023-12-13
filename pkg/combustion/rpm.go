package combustion

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/suse-edge/edge-image-builder/pkg/repo"
	"github.com/suse-edge/edge-image-builder/pkg/rpm"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
)

const (
	userRPMsDir         = "rpms"
	modifyRPMScriptName = "10-rpm-install.sh"
	rpmComponentName    = "RPM"
	combustionBasePath  = "/dev/shm/combustion/config"
)

//go:embed templates/10-rpm-install.sh.tpl
var modifyRPMScript string

func configureRPMs(ctx *image.Context) ([]string, error) {
	if skipRPMconfigre(ctx) {
		log.AuditComponentSkipped(rpmComponentName)
		zap.L().Info("Skipping RPM component. Configuration is not provided")
		return nil, nil
	}

	zap.L().Info("Configuring RPM component...")
	var repoName string
	var packages []string
	if isResolutionNeeded(ctx) {
		zap.L().Info("Begining package dependency resolution...")
		repoPath, pkgList, err := repo.Create(ctx, ctx.CombustionDir)
		if err != nil {
			log.AuditComponentFailed(rpmComponentName)
			return nil, fmt.Errorf("creating rpm repository: %w", err)
		}
		repoName = filepath.Base(repoPath)
		packages = pkgList
	} else {
		rpms, err := rpm.CopyRPMs(generateComponentPath(ctx, userRPMsDir), ctx.CombustionDir)
		if err != nil {
			log.AuditComponentFailed(rpmComponentName)
			return nil, fmt.Errorf("moving single rpm files: %w", err)
		}
		packages = rpms
	}

	script, err := writeRPMScript(ctx, repoName, packages)
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

func skipRPMconfigre(ctx *image.Context) bool {
	pkg := ctx.ImageDefinition.OperatingSystem.Packages

	// rpm from third party repo
	if isComponentConfigured(ctx, userRPMsDir) && len(pkg.AddRepos) > 0 {
		return false
	} else if isComponentConfigured(ctx, userRPMsDir) {
		// standalone rpm
		return false
	}

	// package from PackageHub
	if len(pkg.PKGList) > 0 && pkg.RegCode != "" {
		return false
	}

	// third party pacakge
	if len(pkg.AddRepos) > 0 && len(pkg.PKGList) > 0 {
		return false
	}

	return true
}

func isResolutionNeeded(ctx *image.Context) bool {
	pkg := ctx.ImageDefinition.OperatingSystem.Packages

	// check if no:
	// 1. packages from PackageHub are provided
	// 2. third party packges are provided
	// 3. no third party repos for rpms are provided
	if len(pkg.PKGList) > 0 && pkg.RegCode != "" {
		return true
	} else if len(pkg.AddRepos) > 0 && len(pkg.PKGList) > 0 {
		return true
	} else if len(pkg.AddRepos) > 0 {
		return true
	}
	return false
}

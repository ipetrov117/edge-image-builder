package build

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/template"
)

const (
	zypperRepoScriptName = "create-zypper-repo.sh"
	zypperRepoScriptMode = 0o744
)

//go:embed scripts/rpms/create-zypper-repo.sh.tpl
var zypperRepoScriptTemplate string

// func (b *Builder) createZypperRepoCommand() *exec.Cmd {
// 	scriptPath := filepath.Join(b.context.BuildDir, zypperRepoScriptName)
// 	cmd := exec.Command(scriptPath)
// 	return cmd
// }

func (b *Builder) buildZypperRepo() error {
	err := b.writeZypperRepoScript()
	if err != nil {
		return fmt.Errorf("writing the zypper repo creation script: %w", err)
	}

	time.Sleep(10 * time.Hour)

	// cmd := b.createZypperRepoCommand()
	// err = cmd.Run()
	// if err != nil {
	// 	return fmt.Errorf("running the zypper repo creation script: %w", err)
	// }

	// err = cp.Copy(filepath.Join(b.context.BuildDir, "repo"), b.context.CombustionDir)
	// if err != nil {
	// 	return fmt.Errorf("error copying the zypper repo: %w", err)
	// }

	return nil
}

func (b *Builder) writeZypperRepoScript() error {
	workDir, err := os.MkdirTemp("", "eib")
	if err != nil {
		return fmt.Errorf("creating tmp directory error: %w", err)
	}

	values := struct {
		ISOPath         string
		PKGList         string
		AdditionalRepos string
		RegCode         string
		WorkDir         string
	}{
		ISOPath:         b.generateBaseImageFilename(),
		PKGList:         strings.Join(b.imageConfig.OperatingSystem.SUSEPackages.PKGList, " "),
		AdditionalRepos: strings.Join(b.imageConfig.OperatingSystem.SUSEPackages.Repos, " "),
		RegCode:         b.imageConfig.OperatingSystem.SUSEPackages.RegCode,
		WorkDir:         workDir,
	}

	data, err := template.Parse(zypperRepoScriptName, zypperRepoScriptTemplate, &values)
	if err != nil {
		return fmt.Errorf("parsing %s template: %w", zypperRepoScriptName, err)
	}

	filename := b.generateBuildDirFilename(zypperRepoScriptName)
	if err = os.WriteFile(filename, []byte(data), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing modification script %s: %w", zypperRepoScriptName, err)
	}

	return nil
}

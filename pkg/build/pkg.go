package build

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/template"
)

const (
	zypperRepoScriptName  = "create-zypper-repo.sh"
	zypperRepoWorkDirName = "air-gapped-repo"
)

//go:embed scripts/rpms/create-zypper-repo.sh.tpl
var zypperRepoScriptTemplate string

func (b *Builder) buildZypperRepo() error {
	if err := b.prepareRepoEnv(); err != nil {
		return fmt.Errorf("preparing zypper repo work directory: %w", err)
	}

	if err := b.writeZypperRepoScript(); err != nil {
		return fmt.Errorf("writing the zypper repo creation script: %w", err)
	}

	time.Sleep(10 * time.Hour)

	cmd := b.createZypperRepoCacheCmd()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running the zypper repo creation script: %w", err)
	}

	// if err := fileio.CopyDir(filepath.Join(b.generateRepoWorkDirPath(), "repo"))

	// if err := fileio.CopyDir()
	// err = cp.Copy(filepath.Join(b.context.BuildDir, workDirName, "repo"), filepath.Join(b.context.CombustionDir, "repo"))
	// if err != nil {
	// 	return fmt.Errorf("error copying the zypper repo: %w", err)
	// }

	return nil
}

func (b *Builder) prepareRepoEnv() error {
	wdPath := b.generateRepoWorkDirPath()
	rootPath := filepath.Join(wdPath, "root")
	repoPath := b.generateRepoPath()

	if err := os.RemoveAll(wdPath); err != nil {
		return fmt.Errorf("removing previous repo work directory: %w", err)
	}

	if err := os.MkdirAll(rootPath, os.ModePerm); err != nil {
		return fmt.Errorf("creating %s dir in work directory %s: %w", rootPath, wdPath, err)
	}

	if err := os.MkdirAll(repoPath, os.ModePerm); err != nil {
		return fmt.Errorf("creating zypper repo dir in %s: %w", repoPath, err)
	}

	if err := fileio.CopyFile(b.generateBaseImageFilename(), filepath.Join(wdPath, b.imageConfig.Image.BaseImage), fileio.NonExecutablePerms); err != nil {
		return fmt.Errorf("creating work copy of image %s in repo work dir %s: %w", b.generateBaseImageFilename(), wdPath, err)
	}

	return nil
}

func (b *Builder) writeZypperRepoScript() error {
	values := struct {
		ISOName         string
		PKGList         string
		AdditionalRepos string
		RegCode         string
		WorkDir         string
		RepoOut         string
	}{
		ISOName:         b.imageConfig.Image.BaseImage,
		PKGList:         strings.Join(b.imageConfig.OperatingSystem.SUSEPackages.PKGList, " "),
		AdditionalRepos: strings.Join(b.imageConfig.OperatingSystem.SUSEPackages.Repos, " "),
		RegCode:         b.imageConfig.OperatingSystem.SUSEPackages.RegCode,
		WorkDir:         b.generateRepoWorkDirPath(),
		RepoOut:         b.generateRepoPath(),
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

func (b *Builder) generateRepoWorkDirPath() string {
	return filepath.Join(b.context.BuildDir, zypperRepoWorkDirName)
}

func (b *Builder) generateRepoPath() string {
	return filepath.Join(b.context.CombustionDir, zypperRepoWorkDirName)
}

func (b *Builder) createZypperRepoCacheCmd() *exec.Cmd {
	scriptPath := filepath.Join(b.context.BuildDir, zypperRepoScriptName)
	cmd := exec.Command(scriptPath)
	return cmd
}

func (b *Builder) createZypperRepo() error {

	return nil
}

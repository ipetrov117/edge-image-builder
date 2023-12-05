package resolver

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/template"
)

const (
	prepareBaseScriptName = "prepare-base.sh"
	baseImageArchiveName  = "sle-micro-base.tar.gz"
	baseImageRef          = "slemicro"
	dockerfileName        = "Dockerfile"
)

//go:embed scripts/prepare-base.sh.tpl
var prepareBaseTemplate string

//go:embed scripts/Dockerfile.tpl
var dockerfileTemplate string

func (p *PackageResolver) buildImage(name string) error {
	if err := p.buildBaseImage(); err != nil {
		return fmt.Errorf("building base resolver image: %w", err)
	}

	if err := p.writeDockerfile(); err != nil {
		return fmt.Errorf("writing dockerfile: %w", err)
	}

	if err := p.podman.Build(p.generateResolverDirPath(), name); err != nil {
		return fmt.Errorf("building resolver image: %w", err)
	}

	return nil
}

func (p *PackageResolver) buildBaseImage() error {
	defer os.RemoveAll(p.generateBaseImageDirPath())

	if err := p.prepareBaseImage(); err != nil {
		return fmt.Errorf("preparing base resolver image: %w", err)
	}

	if err := p.writeBaseImageScript(); err != nil {
		return fmt.Errorf("writing base resolver image script: %w", err)
	}

	cmd := p.prepareBaseImageCommand()
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("running the prepare base image script: %w", err)
	}

	tarballPath := filepath.Join(p.generateBaseImageDirPath(), baseImageArchiveName)
	_, err = p.podman.Import(tarballPath, baseImageRef)
	if err != nil {
		return fmt.Errorf("importing the base image: %w", err)
	}

	return nil
}

func (p *PackageResolver) prepareBaseImage() error {
	baseImgDir := p.generateBaseImageDirPath()

	if err := os.RemoveAll(baseImgDir); err != nil {
		return fmt.Errorf("removing previous repo work directory: %w", err)
	}

	if err := os.MkdirAll(baseImgDir, os.ModePerm); err != nil {
		return fmt.Errorf("creating %s dir: %w", baseImgDir, err)
	}

	originalImgPath := filepath.Join(p.ctx.ImageConfigDir, "images", p.ctx.ImageDefinition.Image.BaseImage)
	if err := fileio.CopyFile(originalImgPath, filepath.Join(baseImgDir, p.ctx.ImageDefinition.Image.BaseImage), fileio.NonExecutablePerms); err != nil {
		return fmt.Errorf("creating work copy of image %s in repo work dir %s: %w", originalImgPath, baseImgDir, err)
	}

	return nil
}

func (p *PackageResolver) writeBaseImageScript() error {
	baseImgDir := p.generateBaseImageDirPath()

	values := struct {
		BaseImageDir string
		BaseISOPath  string
		ArchiveName  string
	}{
		BaseImageDir: baseImgDir,
		BaseISOPath:  filepath.Join(baseImgDir, p.ctx.ImageDefinition.Image.BaseImage),
		ArchiveName:  baseImageArchiveName,
	}

	data, err := template.Parse(prepareBaseScriptName, prepareBaseTemplate, &values)
	if err != nil {
		return fmt.Errorf("parsing %s template: %w", prepareBaseScriptName, err)
	}

	filename := filepath.Join(p.ctx.BuildDir, prepareBaseScriptName)
	if err = os.WriteFile(filename, []byte(data), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing prepare base image script %s: %w", filename, err)
	}

	return nil
}

func (p *PackageResolver) prepareBaseImageCommand() *exec.Cmd {
	scriptPath := filepath.Join(p.ctx.BuildDir, prepareBaseScriptName)
	cmd := exec.Command(scriptPath)
	return cmd
}

func (p *PackageResolver) writeDockerfile() error {
	values := struct {
		BaseImage string
		RegCode   string
		AddRepo   string
		CacheDir  string
		PkgList   string
	}{
		BaseImage: baseImageRef,
		RegCode:   p.ctx.ImageDefinition.OperatingSystem.SUSEPackages.RegCode,
		AddRepo:   strings.Join(p.ctx.ImageDefinition.OperatingSystem.SUSEPackages.AddRepos, " "),
		CacheDir:  p.generateRepoDirPathInImage(),
		PkgList:   strings.Join(p.generatePackageList(), " "),
	}

	data, err := template.Parse(dockerfileName, dockerfileTemplate, &values)
	if err != nil {
		return fmt.Errorf("parsing %s template: %w", dockerfileName, err)
	}

	filename := filepath.Join(p.generateResolverDirPath(), dockerfileName)
	if err = os.WriteFile(filename, []byte(data), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing prepare base image script %s: %w", filename, err)
	}

	return nil
}

package resolver

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/podman"
	"github.com/suse-edge/edge-image-builder/pkg/template"
)

const (
	prepareBaseScriptName = "prepare-base.sh"
	internalImageDirName  = "resolver-image-base"
	resolverDirName       = "resolver"
	baseImageArchiveName  = "sle-micro-base.tar.gz"
	baseImageRef          = "slemicro"
	resolverImageRef      = "pkg-resolver"
	dockerfileName        = "Dockerfile"
)

//go:embed scripts/prepare-base.sh.tpl
var prepareBaseTemplate string

//go:embed scripts/Dockerfile.tpl
var dockerfileTemplate string

type imageBuilder struct {
	client  *podman.Podman
	context *image.Context
}

func (r *imageBuilder) buildResolverImage() error {
	if err := r.buildBaseImage(); err != nil {
		return fmt.Errorf("building base resolver image: %w", err)
	}

	if err := r.writeDockerfile(); err != nil {
		return fmt.Errorf("writing dockerfile: %w", err)
	}

	if err := r.client.Build(r.generateResolverDirPath(), resolverImageRef); err != nil {
		return fmt.Errorf("building resolver image: %w", err)
	}

	// TODO:
	// 1. Build image
	// 2. Run container from image
	// 3. Testout everything works
	// 4. Populate the Dockerfile with the correct zypper commands
	// 5. Copy the repo out of the container in the resolver
	// 6. Call resolver from rpm.go and rename it to pkg.go
	return nil
}

func (r *imageBuilder) buildBaseImage() error {
	defer os.RemoveAll(r.generateBaseImageDirPath())

	if err := r.prepareBaseImage(); err != nil {
		return fmt.Errorf("preparing base resolver image: %w", err)
	}

	if err := r.writeBaseImageScript(); err != nil {
		return fmt.Errorf("writing base resolver image script: %w", err)
	}

	cmd := r.prepareBaseImageCommand()
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("running the prepare base image script: %w", err)
	}

	tarballPath := filepath.Join(r.generateBaseImageDirPath(), baseImageArchiveName)
	_, err = r.client.Import(tarballPath, baseImageRef)
	if err != nil {
		return fmt.Errorf("importing the base image: %w", err)
	}

	return nil
}

func (r *imageBuilder) prepareBaseImage() error {
	baseImgDir := r.generateBaseImageDirPath()

	if err := os.RemoveAll(baseImgDir); err != nil {
		return fmt.Errorf("removing previous repo work directory: %w", err)
	}

	if err := os.MkdirAll(baseImgDir, os.ModePerm); err != nil {
		return fmt.Errorf("creating %s dir: %w", baseImgDir, err)
	}

	originalImgPath := filepath.Join(r.context.ImageConfigDir, "images", r.context.ImageDefinition.Image.BaseImage)
	if err := fileio.CopyFile(originalImgPath, filepath.Join(baseImgDir, r.context.ImageDefinition.Image.BaseImage), fileio.NonExecutablePerms); err != nil {
		return fmt.Errorf("creating work copy of image %s in repo work dir %s: %w", originalImgPath, baseImgDir, err)
	}

	return nil
}

func (r *imageBuilder) writeBaseImageScript() error {
	baseImgDir := r.generateBaseImageDirPath()

	values := struct {
		BaseImageDir string
		BaseISOPath  string
		ArchiveName  string
	}{
		BaseImageDir: baseImgDir,
		BaseISOPath:  filepath.Join(baseImgDir, r.context.ImageDefinition.Image.BaseImage),
		ArchiveName:  baseImageArchiveName,
	}

	data, err := template.Parse(prepareBaseScriptName, prepareBaseTemplate, &values)
	if err != nil {
		return fmt.Errorf("parsing %s template: %w", prepareBaseScriptName, err)
	}

	filename := filepath.Join(r.context.BuildDir, prepareBaseScriptName)
	if err = os.WriteFile(filename, []byte(data), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing prepare base image script %s: %w", filename, err)
	}

	return nil
}

func (r *imageBuilder) generateResolverDirPath() string {
	return filepath.Join(r.context.BuildDir, resolverDirName)
}

func (r *imageBuilder) generateBaseImageDirPath() string {
	return filepath.Join(r.generateResolverDirPath(), internalImageDirName)
}

func (r *imageBuilder) prepareBaseImageCommand() *exec.Cmd {
	scriptPath := filepath.Join(r.context.BuildDir, prepareBaseScriptName)
	cmd := exec.Command(scriptPath)
	return cmd
}

func (r *imageBuilder) writeDockerfile() error {
	values := struct {
		BaseImage string
	}{
		BaseImage: baseImageRef,
	}

	data, err := template.Parse(dockerfileName, dockerfileTemplate, &values)
	if err != nil {
		return fmt.Errorf("parsing %s template: %w", dockerfileName, err)
	}

	filename := filepath.Join(r.generateResolverDirPath(), dockerfileName)
	if err = os.WriteFile(filename, []byte(data), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing prepare base image script %s: %w", filename, err)
	}

	return nil
}

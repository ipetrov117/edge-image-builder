package resolver

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
)

const (
	prepareBaseScriptName = "prepare-base.sh"
	baseImageArchiveName  = "sle-micro-base.tar.gz"
	baseImageRef          = "slemicro"
)

//go:embed scripts/prepare-base.sh.tpl
var prepareBaseTemplate string

func (r *Resolver) buildBase() error {
	zap.L().Info("Building base resolver image...")

	defer os.RemoveAll(r.getBaseImgDir())
	if err := r.prepareBase(); err != nil {
		return fmt.Errorf("preparing base image env: %w", err)
	}

	if err := r.writeBaseImageScript(); err != nil {
		return fmt.Errorf("writing base resolver image script: %w", err)
	}

	cmd := r.prepareBaseImageCmd()
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("running the prepare base image script: %w", err)
	}

	tarballPath := filepath.Join(r.getBaseImgDir(), baseImageArchiveName)
	_, err = r.podman.Import(tarballPath, baseImageRef)
	if err != nil {
		return fmt.Errorf("importing the base image: %w", err)
	}

	zap.L().Info("Base resolver image build successful")
	return nil
}

func (r *Resolver) prepareBase() error {
	baseImgDir := r.getBaseImgDir()
	if err := os.MkdirAll(baseImgDir, os.ModePerm); err != nil {
		return fmt.Errorf("creating %s dir: %w", baseImgDir, err)
	}

	if err := fileio.CopyFile(r.ImgPath, r.getBaseISOCopyPath(), fileio.NonExecutablePerms); err != nil {
		return fmt.Errorf("creating work copy of image %s in repo work dir %s: %w", r.ImgPath, baseImgDir, err)
	}

	return nil
}

func (r *Resolver) writeBaseImageScript() error {
	values := struct {
		WorkDir     string
		ImgPath     string
		ArchiveName string
		ImgType     string
	}{
		WorkDir:     r.getBaseImgDir(),
		ImgPath:     r.getBaseISOCopyPath(),
		ArchiveName: baseImageArchiveName,
		ImgType:     r.ImgType,
	}

	data, err := template.Parse(prepareBaseScriptName, prepareBaseTemplate, &values)
	if err != nil {
		return fmt.Errorf("parsing %s template: %w", prepareBaseScriptName, err)
	}

	filename := filepath.Join(r.Dir, prepareBaseScriptName)
	if err = os.WriteFile(filename, []byte(data), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing prepare base image script %s: %w", filename, err)
	}

	return nil
}

func (r *Resolver) prepareBaseImageCmd() *exec.Cmd {
	scriptPath := filepath.Join(r.Dir, prepareBaseScriptName)
	cmd := exec.Command(scriptPath)
	return cmd
}

func (r *Resolver) getBaseImgDir() string {
	return filepath.Join(r.Dir, "base-image")
}

func (r *Resolver) getBaseISOCopyPath() string {
	return filepath.Join(r.getBaseImgDir(), filepath.Base(r.ImgPath))
}

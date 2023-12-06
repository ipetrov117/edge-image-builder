package repo

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/podman"
	"github.com/suse-edge/edge-image-builder/pkg/template"
)

const (
	prepareBaseScriptName = "prepare-base.sh"
	baseImageArchiveName  = "sle-micro-base.tar.gz"
	baseImageRef          = "slemicro"
	resolverImageRef      = "pkg-resolver"
	dockerfileName        = "Dockerfile"
)

//go:embed scripts/prepare-base.sh.tpl
var prepareBaseTemplate string

//go:embed scripts/Dockerfile.tpl
var dockerfileTemplate string

type PkgResolver struct {
	Context      string
	BaseISO      string
	PkgToInstall *image.Packages
	RpmDir       string
	Podman       *podman.Podman
	rpmFileNames []string
}

func (r *PkgResolver) Resolve(out string) (string, []string, error) {
	if err := r.buildBase(); err != nil {
		return "", nil, fmt.Errorf("building base resolver image: %w", err)
	}

	if err := r.prepare(); err != nil {
		return "", nil, fmt.Errorf("generating context for the resolver image: %w", err)
	}

	if err := r.Podman.Build(r.getImgDir(), resolverImageRef); err != nil {
		return "", nil, fmt.Errorf("building resolver image: %w", err)
	}

	id, err := r.Podman.Run(resolverImageRef)
	if err != nil {
		return "", nil, fmt.Errorf("run container from resolver image %s: %w", resolverImageRef, err)
	}

	err = r.Podman.Copy(id, r.getPkgCacheDirInImage(), out)
	if err != nil {
		return "", nil, fmt.Errorf("copying resolved pkg cache to %s: %w", out, err)
	}

	return filepath.Base(r.getPkgCacheDirInImage()), r.getPkgList(), nil
}

func (r *PkgResolver) buildBase() error {
	baseImgDir := r.getBaseImgDir()
	if err := os.MkdirAll(baseImgDir, os.ModePerm); err != nil {
		return fmt.Errorf("creating %s dir: %w", baseImgDir, err)
	}
	defer os.RemoveAll(baseImgDir)

	originalImgPath := filepath.Join(r.BaseISO)
	if err := fileio.CopyFile(originalImgPath, r.getBaseISOCopyPath(), fileio.NonExecutablePerms); err != nil {
		return fmt.Errorf("creating work copy of image %s in repo work dir %s: %w", originalImgPath, baseImgDir, err)
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
	_, err = r.Podman.Import(tarballPath, baseImageRef)
	if err != nil {
		return fmt.Errorf("importing the base image: %w", err)
	}

	return nil
}

func (r *PkgResolver) writeBaseImageScript() error {
	values := struct {
		BaseImageDir string
		BaseISOPath  string
		ArchiveName  string
	}{
		BaseImageDir: r.getBaseImgDir(),
		BaseISOPath:  r.getBaseISOCopyPath(),
		ArchiveName:  baseImageArchiveName,
	}

	data, err := template.Parse(prepareBaseScriptName, prepareBaseTemplate, &values)
	if err != nil {
		return fmt.Errorf("parsing %s template: %w", prepareBaseScriptName, err)
	}

	filename := filepath.Join(r.Context, prepareBaseScriptName)
	if err = os.WriteFile(filename, []byte(data), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing prepare base image script %s: %w", filename, err)
	}

	return nil
}

func (r *PkgResolver) prepareBaseImageCmd() *exec.Cmd {
	scriptPath := filepath.Join(r.Context, prepareBaseScriptName)
	cmd := exec.Command(scriptPath)
	return cmd
}

func (r *PkgResolver) prepare() error {
	if err := r.prepareRPMNames(); err != nil {
		return fmt.Errorf("preparing rpm name slice: %w", err)
	}

	imgDir := r.getImgDir()
	if err := os.MkdirAll(imgDir, fileio.NonExecutablePerms); err != nil {
		return fmt.Errorf("creating build context dir %s: %w", imgDir, err)
	}

	if err := r.copyRPMsToImgDir(); err != nil {
		return fmt.Errorf("copying local rpms to resolver image dir: %w", err)
	}

	if err := r.writeDockerfile(); err != nil {
		return fmt.Errorf("writing dockerfile: %w", err)
	}
	return nil
}

func (r *PkgResolver) prepareRPMNames() error {
	rpms, err := os.ReadDir(r.RpmDir)
	if err != nil {
		return fmt.Errorf("reading RPM source dir: %w", err)
	}

	for _, rpmFile := range rpms {
		if filepath.Ext(rpmFile.Name()) == ".rpm" {
			r.rpmFileNames = append(r.rpmFileNames, rpmFile.Name())
		}
	}
	return nil
}

func (r *PkgResolver) copyRPMsToImgDir() error {
	exists, err := fileExists(r.RpmDir)
	if err != nil {
		return fmt.Errorf("checking if %s dir exists: %w", r.RpmDir, err)
	}

	if exists {
		dest := r.getImgDirRPMPath()
		if err := os.MkdirAll(dest, os.ModePerm); err != nil {
			return fmt.Errorf("creating rpm dir in build context")
		}
		if err := copyRPMs(r.RpmDir, dest, r.rpmFileNames); err != nil {
			return fmt.Errorf("copying local rpms to docker build dir %s: %w", dest, err)
		}
	}

	return nil
}

func (r *PkgResolver) writeDockerfile() error {
	values := struct {
		BaseImage   string
		RegCode     string
		AddRepo     string
		CacheDir    string
		PkgList     string
		FromRPMPath string
		ToRPMPath   string
	}{
		BaseImage:   baseImageRef,
		RegCode:     r.PkgToInstall.RegCode,
		AddRepo:     strings.Join(r.PkgToInstall.AddRepos, " "),
		CacheDir:    r.getPkgCacheDirInImage(),
		PkgList:     strings.Join(r.getPkgList(), " "),
		FromRPMPath: r.getImgDirRPMPath(),
		ToRPMPath:   r.getLocalRPMDirInImage(),
	}

	data, err := template.Parse(dockerfileName, dockerfileTemplate, &values)
	if err != nil {
		return fmt.Errorf("parsing %s template: %w", dockerfileName, err)
	}

	filename := filepath.Join(r.getImgDir(), dockerfileName)
	if err = os.WriteFile(filename, []byte(data), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing prepare base image script %s: %w", filename, err)
	}

	return nil
}

func (r *PkgResolver) getPkgList() []string {
	fullList := []string{}
	pkgList := r.PkgToInstall.PKGList
	if len(pkgList) > 0 {
		fullList = append(fullList, pkgList...)
	}

	for _, rpm := range r.rpmFileNames {
		fullList = append(fullList, filepath.Join(r.getImgDirRPMPath(), rpm))
	}

	return fullList
}

func (r *PkgResolver) getBaseImgDir() string {
	return filepath.Join(r.Context, "resolver-image-base")
}

func (r *PkgResolver) getBaseISOCopyPath() string {
	return filepath.Join(r.getBaseImgDir(), filepath.Base(r.BaseISO))
}

func (r *PkgResolver) getImgDir() string {
	return filepath.Join(r.Context, "build")
}

func (r *PkgResolver) getImgDirRPMPath() string {
	return filepath.Join(r.getImgDir(), "rpms")
}

func (r *PkgResolver) getPkgCacheDirInImage() string {
	return filepath.Join(os.TempDir(), "pkg-cache")
}

func (r *PkgResolver) getLocalRPMDirInImage() string {
	return filepath.Join(r.getPkgCacheDirInImage(), "local")
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("describing file at %s: %w", path, err)
	}
	return true, nil
}

func copyRPMs(rpmSourceDir string, rpmDestDir string, rpmFileNames []string) error {
	if rpmDestDir == "" {
		return fmt.Errorf("RPM destination directory cannot be empty")
	}

	for _, rpm := range rpmFileNames {
		sourcePath := filepath.Join(rpmSourceDir, rpm)
		destPath := filepath.Join(rpmDestDir, rpm)

		err := fileio.CopyFile(sourcePath, destPath, fileio.NonExecutablePerms)
		if err != nil {
			return fmt.Errorf("copying file %s: %w", sourcePath, err)
		}
	}

	return nil
}

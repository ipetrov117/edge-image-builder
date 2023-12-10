package resolver

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/podman"
	"github.com/suse-edge/edge-image-builder/pkg/template"
)

const (
	resolverImageRef = "pkg-resolver"
	dockerfileName   = "Dockerfile"
)

//go:embed scripts/Dockerfile.tpl
var dockerfileTemplate string

type Resolver struct {
	dir          string
	imgPath      string
	packages     *image.Packages
	customRPMDir string
	podman       *podman.Podman
	rpmNames     []string
}

func New(dir, usrImg, rpmDir string, packages *image.Packages) (*Resolver, error) {
	p, err := podman.New(dir)
	if err != nil {
		return nil, fmt.Errorf("starting podman client: %w", err)
	}

	resolverDir := filepath.Join(dir, "resolver")
	if err := os.MkdirAll(resolverDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("creating %s dir: %w", resolverDir, err)
	}

	resolver := &Resolver{
		dir:          resolverDir,
		imgPath:      usrImg, //filepath.Join(ctx.ImageConfigDir, "images", ctx.ImageDefinition.Image.BaseImage),
		packages:     packages,
		podman:       p,
		customRPMDir: rpmDir,
	}
	return resolver, nil
}

func (r *Resolver) Resolve(out string) (string, []string, error) {
	if err := r.buildBase(); err != nil {
		return "", nil, fmt.Errorf("building base resolver image: %w", err)
	}

	if err := r.prepare(); err != nil {
		return "", nil, fmt.Errorf("generating context for the resolver image: %w", err)
	}

	if err := r.podman.Build(r.generateBuildContextPath(), resolverImageRef); err != nil {
		return "", nil, fmt.Errorf("building resolver image: %w", err)
	}

	id, err := r.podman.Run(resolverImageRef)
	if err != nil {
		return "", nil, fmt.Errorf("run container from resolver image %s: %w", resolverImageRef, err)
	}

	err = r.podman.Copy(id, r.generateRPMRepoPath(), out)
	if err != nil {
		return "", nil, fmt.Errorf("copying resolved pkg cache to %s: %w", out, err)
	}

	return filepath.Base(r.generateRPMRepoPath()), r.generateReadyToUsePKGList(), nil
}

func (r *Resolver) prepare() error {
	buildContext := r.generateBuildContextPath()
	if err := os.MkdirAll(buildContext, fileio.NonExecutablePerms); err != nil {
		return fmt.Errorf("creating build context dir %s: %w", buildContext, err)
	}

	if r.customRPMDir != "" {
		dest := r.generateRPMPathInBuildContext()
		if err := os.MkdirAll(dest, os.ModePerm); err != nil {
			return fmt.Errorf("creating rpm directory in resolver dir: %w", err)
		}

		rpmNames, err := copyRPMs(r.customRPMDir, dest)
		if err != nil {
			return fmt.Errorf("copying local rpms to %s: %w", dest, err)
		}
		r.rpmNames = rpmNames
	}

	if err := r.writeDockerfile(); err != nil {
		return fmt.Errorf("writing dockerfile: %w", err)
	}

	return nil
}

func (r *Resolver) writeDockerfile() error {
	values := struct {
		BaseImage   string
		RegCode     string
		AddRepo     string
		CacheDir    string
		PkgList     string
		FromRPMPath string
		ToRPMPath   string
	}{
		BaseImage: baseImageRef,
		RegCode:   r.packages.RegCode,
		AddRepo:   strings.Join(r.packages.AddRepos, " "),
		CacheDir:  r.generateRPMRepoPath(),
		PkgList:   strings.Join(r.generateInstallationPKGList(), " "),
	}

	if r.customRPMDir != "" {
		values.FromRPMPath = filepath.Base(r.generateRPMPathInBuildContext())
		values.ToRPMPath = r.generateLocalRPMPath()
	}

	data, err := template.Parse(dockerfileName, dockerfileTemplate, &values)
	if err != nil {
		return fmt.Errorf("parsing %s template: %w", dockerfileName, err)
	}

	filename := filepath.Join(r.generateBuildContextPath(), dockerfileName)
	if err = os.WriteFile(filename, []byte(data), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing prepare base image script %s: %w", filename, err)
	}

	return nil
}

func (r *Resolver) generateInstallationPKGList() []string {
	list := []string{}

	if len(r.packages.PKGList) > 0 {
		list = append(list, r.packages.PKGList...)
	}

	for _, name := range r.rpmNames {
		list = append(list, filepath.Join(r.generateLocalRPMPath(), name))
	}
	return list
}

func (r *Resolver) generateReadyToUsePKGList() []string {
	list := []string{}

	if len(r.packages.PKGList) > 0 {
		list = append(list, r.packages.PKGList...)
	}

	for _, name := range r.rpmNames {
		list = append(list, strings.TrimSuffix(name, filepath.Ext(name)))
	}
	return list
}

func (r *Resolver) generateBuildContextPath() string {
	return filepath.Join(r.dir, "build")
}

func (r *Resolver) generateRPMPathInBuildContext() string {
	return filepath.Join(r.generateBuildContextPath(), "rpms")
}

func (r *Resolver) generateRPMRepoPath() string {
	return filepath.Join(os.TempDir(), "rpm-repo")
}

func (r *Resolver) generateLocalRPMPath() string {
	return filepath.Join(r.generateRPMRepoPath(), "local")
}

func copyRPMs(rpmSourceDir string, rpmDestDir string) ([]string, error) {
	list := []string{}

	rpms, err := os.ReadDir(rpmSourceDir)
	if err != nil {
		return nil, fmt.Errorf("reading RPM source dir: %w", err)
	}

	for _, rpm := range rpms {
		if filepath.Ext(rpm.Name()) == ".rpm" {
			sourcePath := filepath.Join(rpmSourceDir, rpm.Name())
			destPath := filepath.Join(rpmDestDir, rpm.Name())

			err := fileio.CopyFile(sourcePath, destPath, fileio.NonExecutablePerms)
			if err != nil {
				return nil, fmt.Errorf("copying file %s: %w", sourcePath, err)
			}
			list = append(list, rpm.Name())
		}
	}

	return list, nil
}

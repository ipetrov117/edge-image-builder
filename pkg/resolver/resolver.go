package resolver

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/podman"
)

const (
	imageName = "pkg-resolver"
	repoName  = "pkg-repo"
)

type PackageResolver struct {
	ctx    *image.Context
	podman *podman.Podman
}

func New(ctx *image.Context) (*PackageResolver, error) {
	p, err := podman.New(ctx.BuildDir)
	if err != nil {
		return nil, fmt.Errorf("starting podman client: %w", err)
	}

	return &PackageResolver{
		ctx:    ctx,
		podman: p,
	}, nil
}

func (p *PackageResolver) Resolve() (string, error) {
	cID, err := p.runResolverContainer()
	if err != nil {
		return "", fmt.Errorf("running resolver container: %w", err)
	}

	repoPath := p.ctx.CombustionDir
	err = p.podman.Copy(cID, p.generateRepoDirPathInImage(), repoPath)
	if err != nil {
		return "", err
	}

	return filepath.Join(repoPath, repoName), nil
}

func (p *PackageResolver) runResolverContainer() (string, error) {
	if err := p.buildImage(imageName); err != nil {
		return "", fmt.Errorf("building resolver image %s: %w", imageName, err)
	}

	id, err := p.podman.Run(imageName)
	if err != nil {
		return "", fmt.Errorf("run container from resolver image %s: %w", imageName, err)
	}

	return id, nil
}

func (p *PackageResolver) generatePackageList() []string {
	pkgList := []string{}
	susePkg := p.ctx.ImageDefinition.OperatingSystem.SUSEPackages.PKGList
	if len(susePkg) > 0 {
		pkgList = append(pkgList, susePkg...)
	}

	// TODO: do local RPM logic here

	return pkgList
}

func (p *PackageResolver) generateResolverDirPath() string {
	return filepath.Join(p.ctx.BuildDir, "resolver")
}

func (p *PackageResolver) generateBaseImageDirPath() string {
	return filepath.Join(p.generateResolverDirPath(), "resolver-image-base")
}

func (p *PackageResolver) generateRepoDirPathInImage() string {
	return filepath.Join(os.TempDir(), repoName)
}

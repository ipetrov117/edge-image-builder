package podman

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containers/buildah/define"
	"github.com/containers/podman/v4/pkg/bindings"
	"github.com/containers/podman/v4/pkg/bindings/containers"
	"github.com/containers/podman/v4/pkg/bindings/images"
	"github.com/containers/podman/v4/pkg/domain/entities"
	"github.com/containers/podman/v4/pkg/specgen"
)

const (
	podmanSock         = "unix:///run/podman/podman.sock"
	dockerfile         = "Dockerfile"
	podmanDirName      = "podman"
	podmanBuildLogFile = "podman-image-build-%s.log"
)

type Podman struct {
	context context.Context
	socket  string
	out     string
}

func New(out string) (*Podman, error) {
	podmanDirPath := filepath.Join(out, podmanDirName)
	if err := os.MkdirAll(podmanDirPath, os.ModePerm); err != nil {
		return nil, fmt.Errorf("creating %s dir: %w", podmanDirPath, err)
	}

	if err := setupAPIListener(podmanDirPath); err != nil {
		return nil, fmt.Errorf("creating new podman instance: %w", err)
	}

	conn, err := bindings.NewConnection(context.Background(), podmanSock)
	if err != nil {
		return nil, fmt.Errorf("creating new podman connection: %w", err)
	}

	return &Podman{
		context: conn,
		socket:  podmanSock,
		out:     podmanDirPath,
	}, nil
}

func (p *Podman) Import(tarball, ref string) (*entities.ImageImportReport, error) {
	f, err := os.Open(tarball)
	if err != nil {
		return nil, fmt.Errorf("opening tarball %s: %w", tarball, err)
	}
	report, err := images.Import(p.context, f, &images.ImportOptions{Reference: &ref})
	if err != nil {
		return nil, fmt.Errorf("importing tarball %s: %w", tarball, err)
	}

	return report, nil
}

func (p *Podman) Build(context, name string) error {
	logFile, err := generatePodmanLogFile(podmanBuildLogFile, p.out)
	if err != nil {
		return fmt.Errorf("generating podman build log file: %w", err)
	}

	eOpts := entities.BuildOptions{
		BuildOptions: define.BuildOptions{
			ContextDirectory: context,
			Output:           name,
			Out:              logFile,
			Err:              logFile,
		},
	}

	_, err = images.Build(p.context, []string{dockerfile}, eOpts)
	if err != nil {
		return fmt.Errorf("building image from context %s: %w", context, err)
	}

	return nil
}

func (p *Podman) Run(img string) (string, error) {
	s := specgen.NewSpecGenerator(img, false)
	createResponse, err := containers.CreateWithSpec(p.context, s, nil)
	if err != nil {
		return "", fmt.Errorf("creating container with sepc %v: %w", s, err)
	}

	if err := containers.Start(p.context, createResponse.ID, nil); err != nil {
		return "", fmt.Errorf("starting container with sepc %v: %w", s, err)
	}

	return createResponse.ID, nil
}

func (p *Podman) Copy(id, src, dest string) error {
	tmpArchName := "tmp.tar"
	tmpArch, err := os.Create(filepath.Join(os.TempDir(), tmpArchName))
	if err != nil {
		return fmt.Errorf("creating podman log file: %w", err)
	}

	defer os.RemoveAll(tmpArch.Name())

	copyFunc, err := containers.CopyToArchive(p.context, id, src, tmpArch)
	if err != nil {
		return fmt.Errorf("creating copy function for archive from %s: %w", src, err)
	}

	if err := copyFunc(); err != nil {
		return fmt.Errorf("copying archive from %s to %s: %w", src, dest, err)
	}

	if err := untar(tmpArch.Name(), dest); err != nil {
		return fmt.Errorf("extracting archive to %s: %w", dest, err)
	}

	return nil
}

package podman

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containers/buildah/define"
	"github.com/containers/podman/v4/pkg/bindings"
	"github.com/containers/podman/v4/pkg/bindings/images"
	"github.com/containers/podman/v4/pkg/domain/entities"
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

func (p *Podman) Run() error {
	return nil
}

// func (p *Podman) Build(file string) error {
// 	fmt.Println("IN")
// 	// logFile, err := os.Create(filepath.Join("tmp", "test-log-file.txt"))
// 	// if err != nil {
// 	// 	return err
// 	// }

// 	out, err := os.Create(filepath.Join("tmp", "test-out"))
// 	if err != nil {
// 		return fmt.Errorf("opening podman log file: %w", err)
// 	}

// 	err1, err := os.Create(filepath.Join("tmp", "test-err"))
// 	if err != nil {
// 		return fmt.Errorf("opening podman log file: %w", err)
// 	}

// 	report, err := os.Create(filepath.Join("tmp", "test-report"))
// 	if err != nil {
// 		return fmt.Errorf("opening podman log file: %w", err)
// 	}

// 	eOpts := entities.BuildOptions{
// 		BuildOptions: define.BuildOptions{
// 			// ContainerSuffix: "test",
// 			// Registry:        "test-reg",
// 			// Manifest:        "test-mani",
// 			// Output:  "test-out",
// 			// LogFile: logFile.Name(),
// 			ContextDirectory: file,
// 			AdditionalTags:   []string{"test"},
// 			Output:           "test-out",
// 			Out:              out,
// 			Err:              err1,
// 			ReportWriter:     report,
// 		},
// 	}

// 	fmt.Println(file)

// 	_, err = images.Build(p.context, []string{"Dockerfile"}, eOpts)
// 	if err != nil {
// 		return fmt.Errorf("building image from Dockerfile %s: %w", file, err)
// 	}

// 	fmt.Println("OUT")

// 	return nil
// }

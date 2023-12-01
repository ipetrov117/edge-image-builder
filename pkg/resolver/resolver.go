package resolver

import (
	"fmt"

	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/podman"
)

// "github.com/suse-edge/edge-image-builder/pkg/image"

func Resolve(ctx *image.Context) (string, error) {
	c, err := podman.New(ctx.BuildDir)
	if err != nil {
		return "", fmt.Errorf("starting podman client: %w", err)
	}

	ib := &imageBuilder{
		client:  c,
		context: ctx,
	}

	if err := ib.buildResolverImage(); err != nil {
		return "", err
	}

	return "", nil
}

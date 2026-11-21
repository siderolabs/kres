// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Pin architecture-neutral images to a single platform. Avoids a circular dependency between bldr and pkgs when bootstrapping a new architecture.
const archNeutralPlatform = "linux/amd64"

// InputImage provides common input image used to build containers.
type InputImage struct {
	dag.BaseNode

	Image    string
	Version  string
	Platform string
}

// CompileDockerfile implements dockerfile.Compiler.
func (inputImage *InputImage) CompileDockerfile(output *dockerfile.Output) error {
	output.Stage(inputImage.Name()).
		From(inputImage.Image + ":" + inputImage.Version).
		Platform(inputImage.Platform)

	return nil
}

// NewFHS builds standard input image for FHS.
func NewFHS(*meta.Options) *InputImage {
	return &InputImage{
		BaseNode: dag.NewBaseNode("image-fhs"),

		Image:    "ghcr.io/siderolabs/fhs",
		Version:  config.PkgsVersion,
		Platform: archNeutralPlatform,
	}
}

// NewCACerts builds standard input image for ca-certificates.
func NewCACerts(_ *meta.Options) *InputImage {
	return &InputImage{
		BaseNode: dag.NewBaseNode("image-ca-certificates"),

		Image:    "ghcr.io/siderolabs/ca-certificates",
		Version:  config.PkgsVersion,
		Platform: archNeutralPlatform,
	}
}

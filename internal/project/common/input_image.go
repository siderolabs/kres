// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"fmt"

	"github.com/talos-systems/kres/internal/dag"
	"github.com/talos-systems/kres/internal/output/dockerfile"
	"github.com/talos-systems/kres/internal/project/meta"
)

// InputImage provides common input image used to build containers.
type InputImage struct {
	dag.BaseNode

	Image   string
	Version string
}

// CompileDockerfile implements dockerfile.Compiler.
func (inputImage *InputImage) CompileDockerfile(output *dockerfile.Output) error {
	output.Stage(inputImage.Name()).
		From(fmt.Sprintf("%s:%s", inputImage.Image, inputImage.Version))

	return nil
}

// NewFHS builds standard input image for FHS.
func NewFHS(meta *meta.Options) *InputImage {
	return &InputImage{
		BaseNode: dag.NewBaseNode(fmt.Sprintf("image-%s", "fhs")),

		Image:   "ghcr.io/siderolabs/fhs",
		Version: "v1.0.0",
	}
}

// NewCACerts builds standard input image for ca-certificates.
func NewCACerts(meta *meta.Options) *InputImage {
	return &InputImage{
		BaseNode: dag.NewBaseNode(fmt.Sprintf("image-%s", "ca-certificates")),

		Image:   "ghcr.io/siderolabs/ca-certificates",
		Version: "v1.0.0",
	}
}

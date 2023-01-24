// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"path/filepath"

	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Generate provides .proto compilation with grpc-go plugin
// and go generate runner.
type Generate struct {
	dag.BaseNode

	meta *meta.Options

	Files []File `yaml:"files"`
}

// File represents file to be generated (copied or downloaded).
type File struct {
	Source      string `yaml:"source"`
	Destination string `yaml:"destination"`
}

// NewGenerate builds Generate node.
func NewGenerate(meta *meta.Options) *Generate {
	return &Generate{
		BaseNode: dag.NewBaseNode("generate-files"),
		meta:     meta,
	}
}

// CompileDockerfile implements dockerfile.Compiler.
func (generate *Generate) CompileDockerfile(output *dockerfile.Output) error {
	generateStage := output.Stage("generate-files").
		Description("generates static files").
		From("scratch")

	for _, file := range generate.Files {
		generateStage.Step(
			step.Add(file.Source, filepath.Join("/", file.Destination)),
		)
	}

	return nil
}

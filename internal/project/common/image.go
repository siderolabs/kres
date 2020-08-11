// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"fmt"

	"github.com/talos-systems/kres/internal/dag"
	"github.com/talos-systems/kres/internal/output/dockerfile"
	"github.com/talos-systems/kres/internal/output/dockerfile/step"
	"github.com/talos-systems/kres/internal/output/drone"
	"github.com/talos-systems/kres/internal/output/makefile"
	"github.com/talos-systems/kres/internal/project/meta"
)

// Image provides common image build target.
type Image struct {
	dag.BaseNode

	meta *meta.Options

	ImageName      string   `yaml:"imageName"`
	Entrypoint     string   `yaml:"entrypoint"`
	EntrypointArgs []string `yaml:"entrypointArgs"`
	PushLatest     bool     `yaml:"pushLatest"`
}

// NewImage initializes Image.
func NewImage(meta *meta.Options, name string) *Image {
	return &Image{
		BaseNode: dag.NewBaseNode("image-" + name),

		meta: meta,

		ImageName:  name,
		Entrypoint: "/" + name,
		PushLatest: true,
	}
}

// CompileDrone implements drone.Compiler.
func (image *Image) CompileDrone(output *drone.Output) error {
	output.Step(drone.MakeStep(image.Name()).
		DependsOn(dag.GatherMatchingInputNames(image, dag.Implements((*drone.Compiler)(nil)))...),
	)

	output.Step(drone.MakeStep(image.Name()).
		Name(fmt.Sprintf("push-%s", image.ImageName)).
		Environment("PUSH", "true").
		ExceptPullRequest().
		DockerLogin().
		DependsOn(image.Name()),
	)

	if image.PushLatest {
		output.Step(drone.MakeStep(image.Name()).
			Name(fmt.Sprintf("push-%s-latest", image.ImageName)).
			Environment("PUSH", "true").
			Environment("TAG", "latest").
			OnlyOnMaster().
			ExceptPullRequest().
			DockerLogin().
			DependsOn(fmt.Sprintf("push-%s", image.ImageName)),
		)
	}

	return nil
}

// CompileMakefile implements makefile.Compiler.
func (image *Image) CompileMakefile(output *makefile.Output) error {
	output.Target(image.Name()).Description(fmt.Sprintf("build image %s", image.ImageName)).
		Script(fmt.Sprintf(`@$(MAKE) target-$@ TARGET_ARGS="--tag=$(REGISTRY)/$(USERNAME)/%s:$(TAG)"`, image.ImageName)).
		Phony()

	return nil
}

// CompileDockerfile implements dockerfile.Compiler.
func (image *Image) CompileDockerfile(output *dockerfile.Output) error {
	inputs := dag.GatherMatchingInputNames(image, dag.Implements((*dockerfile.Compiler)(nil)))
	if len(inputs) == 0 {
		return fmt.Errorf("no inputs for Image block")
	}

	rootInput := inputs[0]
	otherInputs := inputs[1:]

	stage := output.Stage(image.Name()).
		From(rootInput)

	for _, input := range otherInputs {
		stage.Step(step.Copy("/", "/").From(input))
	}

	stage.Step(step.Entrypoint(image.Entrypoint, image.EntrypointArgs...))

	return nil
}

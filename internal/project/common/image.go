// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"fmt"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/output/drone"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Image provides common image build target.
type Image struct {
	dag.BaseNode

	meta *meta.Options

	ExtraEnvironment map[string]string `yaml:"extraEnvironment"`
	BaseImage        string            `yaml:"baseImage"`
	AdditionalImages []string          `yaml:"additionalImages"`
	CopyFrom         []struct {
		Stage       string `yaml:"stage"`
		Source      string `yaml:"source"`
		Destination string `yaml:"destination"`
	} `yaml:"copyFrom"`
	ImageName      string   `yaml:"imageName"`
	Entrypoint     string   `yaml:"entrypoint"`
	EntrypointArgs []string `yaml:"entrypointArgs"`
	CustomCommands []string `yaml:"customCommands"`
	PushLatest     bool     `yaml:"pushLatest"`
}

// ImageSourceLabel is a docker image label to specify image source.
const ImageSourceLabel = "org.opencontainers.image.source"

// NewImage initializes Image.
func NewImage(meta *meta.Options, name string) *Image {
	return &Image{
		BaseNode: dag.NewBaseNode("image-" + name),

		meta: meta,

		BaseImage:        "scratch",
		AdditionalImages: []string{"fhs", "ca-certificates"},
		ImageName:        name,
		Entrypoint:       "/" + name,
		PushLatest:       true,
	}
}

// CompileDrone implements drone.Compiler.
func (image *Image) CompileDrone(output *drone.Output) error {
	output.Step(drone.MakeStep(image.Name()).
		DependsOn(dag.GatherMatchingInputNames(image, dag.Implements[drone.Compiler]())...),
	)

	step := drone.MakeStep(image.Name()).
		Name(fmt.Sprintf("push-%s", image.ImageName)).
		Environment("PUSH", "true").
		ExceptPullRequest().
		DockerLogin().
		DependsOn(image.Name())

	for k, v := range image.ExtraEnvironment {
		step.Environment(k, v)
	}

	output.Step(step)

	if image.PushLatest {
		step := drone.MakeStep(image.Name(), "TAG=latest").
			Name(fmt.Sprintf("push-%s-latest", image.ImageName)).
			Environment("PUSH", "true").
			OnlyOnBranch(image.meta.MainBranch).
			ExceptPullRequest().
			DockerLogin().
			DependsOn(fmt.Sprintf("push-%s", image.ImageName))

		for k, v := range image.ExtraEnvironment {
			step.Environment(k, v)
		}

		output.Step(step)
	}

	return nil
}

// CompileGitHubWorkflow implements ghworkflow.Compiler.
func (image *Image) CompileGitHubWorkflow(output *ghworkflow.Output) error {
	loginStep := &ghworkflow.Step{
		Name: "Login to registry",
		Uses: fmt.Sprintf("docker/login-action@%s", config.LoginActionVersion),
		With: map[string]string{
			"registry": "ghcr.io",
			"username": "${{ github.repository_owner }}",
			"password": "${{ secrets.GITHUB_TOKEN }}",
		},
	}

	loginStep.ExceptPullRequest()

	pushStep := ghworkflow.MakeStep(image.Name()).
		SetName(fmt.Sprintf("push-%s", image.ImageName)).
		SetEnv("PUSH", "true").
		ExceptPullRequest()

	for k, v := range image.ExtraEnvironment {
		pushStep.SetEnv(k, v)
	}

	steps := []*ghworkflow.Step{
		loginStep,
		ghworkflow.MakeStep(image.Name()),
		pushStep,
	}

	if image.PushLatest {
		pushStep := ghworkflow.MakeStep(image.Name(), "TAG=latest").
			SetName(fmt.Sprintf("push-%s-latest", image.ImageName)).
			SetEnv("PUSH", "true").
			ExceptPullRequest()

		for k, v := range image.ExtraEnvironment {
			pushStep.SetEnv(k, v)
		}

		steps = append(
			steps,
			pushStep,
		)
	}

	output.AddStep("default", steps...)

	return nil
}

// CompileMakefile implements makefile.Compiler.
func (image *Image) CompileMakefile(output *makefile.Output) error {
	output.Target(image.Name()).
		Description(fmt.Sprintf("Builds image for %s.", image.ImageName)).
		Script(fmt.Sprintf(`@$(MAKE) target-$@ TARGET_ARGS="--tag=$(REGISTRY)/$(USERNAME)/%s:$(TAG)"`, image.ImageName)).
		Phony()

	return nil
}

// CompileDockerfile implements dockerfile.Compiler.
func (image *Image) CompileDockerfile(output *dockerfile.Output) error {
	stage := output.Stage(image.Name())

	if image.BaseImage == "scratch" {
		stage.From(image.BaseImage)
	} else {
		output.Stage(fmt.Sprintf("base-%s", image.Name())).
			From(image.BaseImage)

		stage.From(fmt.Sprintf("base-%s", image.Name()))
	}

	for _, command := range image.CustomCommands {
		stage.Step(step.Script(command))
	}

	inputs := dag.GatherMatchingInputs(image, dag.Implements[dockerfile.Compiler]())
	if len(inputs) == 0 {
		return fmt.Errorf("no inputs for Image block")
	}

	for _, addImage := range image.AdditionalImages {
		var input dag.Node

		switch addImage {
		case "fhs":
			input = NewFHS(image.meta)
		case "ca-certificates":
			input = NewCACerts(image.meta)
		default:
			return fmt.Errorf("unsupported additional image %q", addImage)
		}

		if compiler, ok := input.(dockerfile.Compiler); ok {
			if err := compiler.CompileDockerfile(output); err != nil {
				return err
			}
		}

		inputs = append(inputs, input)
	}

	stage.Step(step.Arg("TARGETARCH"))

	for _, input := range inputs {
		if build, ok := input.(dockerfile.CmdCompiler); ok && build.Entrypoint() != "" {
			stage.Step(step.Copy(build.Entrypoint(), image.Entrypoint).From(input.Name()))
		} else {
			stage.Step(step.Copy("/", "/").From(input.Name()))
		}
	}

	for _, copyFrom := range image.CopyFrom {
		stage.Step(step.Copy(stringOr(copyFrom.Source, "/"), stringOr(copyFrom.Destination, "/")).From(copyFrom.Stage))
	}

	if image.meta.GitHubOrganization != "" && image.meta.GitHubRepository != "" {
		stage.Step(step.Label(ImageSourceLabel, fmt.Sprintf("https://github.com/%s/%s", image.meta.GitHubOrganization, image.meta.GitHubRepository)))
	}

	stage.Step(step.Entrypoint(image.Entrypoint, image.EntrypointArgs...))

	return nil
}

func stringOr(s string, def string) string {
	if s == "" {
		return def
	}

	return s
}

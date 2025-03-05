// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"errors"
	"fmt"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/output/dockerignore"
	"github.com/siderolabs/kres/internal/output/drone"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// FixLocalDestLocationsScript moves the local build artifacts from the <os>_<arch> subdirectories to the build output root directory.
//
// This is to revert the behavior of buildkit on multi-platform builds.
//
// As we force buildkit to always do multi-platform builds (via `BUILDKIT_MULTI_PLATFORM=1`), we need this fix to restore old output behavior.
//
// This script is appended to the local output build targets.
const FixLocalDestLocationsScript = `
@PLATFORM=$(PLATFORM) DEST=$(DEST) bash -c '\
  for platform in $$(tr "," "\n" <<< "$$PLATFORM"); do \
    directory="$${platform//\//_}"; \
    if [[ -d "$$DEST/$$directory" ]]; then \
	  echo $$platform; \
      mv -f "$$DEST/$$directory/"* $$DEST; \
      rmdir "$$DEST/$$directory/"; \
    fi; \
  done'
`

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
		Platform    string `yaml:"platform"`
	} `yaml:"copyFrom"`
	DependsOn         []string `yaml:"dependsOn"`
	ImageName         string   `yaml:"imageName"`
	Entrypoint        string   `yaml:"entrypoint"`
	EntrypointArgs    []string `yaml:"entrypointArgs"`
	CustomCommands    []string `yaml:"customCommands"`
	AllowedLocalPaths []string `yaml:"allowedLocalPaths"`
	PushLatest        bool     `yaml:"pushLatest"`
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
		Name("push-"+image.ImageName).
		Environment("PUSH", "true").
		ExceptPullRequest().
		DockerLogin().
		DependsOn(image.Name())

	for k, v := range image.ExtraEnvironment {
		step.Environment(k, v)
	}

	output.Step(step)

	if image.PushLatest {
		step := drone.MakeStep(image.Name(), "IMAGE_TAG=latest").
			Name(fmt.Sprintf("push-%s-latest", image.ImageName)).
			Environment("PUSH", "true").
			OnlyOnBranch(image.meta.MainBranch).
			ExceptPullRequest().
			DockerLogin().
			DependsOn("push-" + image.ImageName)

		for k, v := range image.ExtraEnvironment {
			step.Environment(k, v)
		}

		output.Step(step)
	}

	return nil
}

// CompileGitHubWorkflow implements ghworkflow.Compiler.
func (image *Image) CompileGitHubWorkflow(output *ghworkflow.Output) error {
	loginStep := ghworkflow.Step("Login to registry").
		SetUses("docker/login-action@"+config.LoginActionVersion).
		SetWith("registry", "ghcr.io").
		SetWith("username", "${{ github.repository_owner }}").
		SetWith("password", "${{ secrets.GITHUB_TOKEN }}")

	if err := loginStep.SetConditions("except-pull-request"); err != nil {
		return err
	}

	pushStep := ghworkflow.Step("push-"+image.ImageName).
		SetMakeStep(image.Name()).
		SetEnv("PUSH", "true")

	if err := pushStep.SetConditions("except-pull-request"); err != nil {
		return err
	}

	for k, v := range image.ExtraEnvironment {
		pushStep.SetEnv(k, v)
	}

	steps := []*ghworkflow.JobStep{
		loginStep,
		ghworkflow.Step(image.Name()).SetMakeStep(image.Name()),
		pushStep,
	}

	if image.PushLatest {
		pushStep := ghworkflow.Step(fmt.Sprintf("push-%s-latest", image.ImageName)).
			SetMakeStep(image.Name(), "IMAGE_TAG=latest").
			SetEnv("PUSH", "true")

		if err := pushStep.SetConditions("except-pull-request"); err != nil {
			return err
		}

		pushStep.SetConditionOnlyOnBranch(image.meta.MainBranch)

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
	target := output.Target(image.Name()).
		Description(fmt.Sprintf("Builds image for %s.", image.ImageName)).
		Script(fmt.Sprintf(`@$(MAKE) registry-$@ IMAGE_NAME="%s"`, image.ImageName)).
		Phony()

	for _, dependsOn := range image.DependsOn {
		target.Depends(dependsOn)
	}

	return nil
}

// CompileDockerignore implements dockerignore.Compiler.
func (image *Image) CompileDockerignore(output *dockerignore.Output) error {
	output.
		AllowLocalPath(image.AllowedLocalPaths...)

	return nil
}

// CompileDockerfile implements dockerfile.Compiler.
func (image *Image) CompileDockerfile(output *dockerfile.Output) error {
	stage := output.Stage(image.Name())

	if image.BaseImage == "scratch" {
		stage.From(image.BaseImage)
	} else {
		output.Stage("base-" + image.Name()).
			From(image.BaseImage)

		stage.From("base-" + image.Name())
	}

	for _, command := range image.CustomCommands {
		stage.Step(step.Script(command))
	}

	inputs := dag.GatherMatchingInputs(image, dag.Implements[dockerfile.Compiler]())
	if len(inputs) == 0 {
		return errors.New("no inputs for Image block")
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
		stage.Step(step.Copy(stringOr(copyFrom.Source, "/"), stringOr(copyFrom.Destination, "/")).From(copyFrom.Stage).Platform(copyFrom.Platform))
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

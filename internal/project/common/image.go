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

// signImageScriptPrefix signs an image with the Siderolabs image-signer, which drives a Sigstore
// keyless signature through an interactive Google authentication flow.
//
// The signer runs in a container, so it gets a throwaway DOCKER_CONFIG built from GITHUB_TOKEN
// instead of the host credentials, and 127.0.0.1:8585 is published for the OAuth redirect.
//
// The prefix ends mid-reference, at the registry and username: callers complete it with
// "<image name>:$(IMAGE_TAG)" by concatenation, never through a Go format string, because the
// shell printf below has a %s of its own.
const signImageScriptPrefix = `@test -n "$$GITHUB_TOKEN" || { \
  echo 'GITHUB_TOKEN with write:packages scope must be set: gh auth refresh -s write:packages, then GITHUB_TOKEN=$$(gh auth token) make' $@; \
  exit 1; \
}
@TMP=$$(mktemp -d) && trap 'rm -rf "$$TMP"' EXIT && \
  printf '{"auths":{"$(REGISTRY)":{"username":"x","password":"%s"}}}' "$$GITHUB_TOKEN" > "$$TMP/config.json" && \
  docker run --rm -p 127.0.0.1:8585:8585 \
    -v "$$TMP:/dc:ro" -e DOCKER_CONFIG=/dc \
    $(IMAGE_SIGNER_IMAGE) sign --timeout=15m \
    $(REGISTRY)/$(USERNAME)/`

// Image provides common image build target.
type Image struct {
	dag.BaseNode

	meta *meta.Options

	ExtraEnvironment map[string]string `yaml:"extraEnvironment"`
	Environment      map[string]string `yaml:"environment"`
	BaseImage        string            `yaml:"baseImage"`
	AdditionalImages []string          `yaml:"additionalImages"`
	CopyFrom         []struct {
		Name        string `yaml:"name"`
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

// CompileGitHubWorkflow implements ghworkflow.Compiler.
func (image *Image) CompileGitHubWorkflow(output *ghworkflow.Output) error {
	loginStep := ghworkflow.Step("Login to registry").
		SetUsesWithComment(
			"docker/login-action@"+config.LoginActionRef,
			"version: "+config.LoginActionVersion,
		).
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

	output.AddStep(ghworkflow.DefaultJobName, steps...)

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

	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable("IMAGE_SIGNER_IMAGE", "ghcr.io/siderolabs/image-signer:"+config.ImageSignerVersion))

	// signing is always driven by a human going through the Google authentication flow, so this
	// target is never wired into CI or into any other target.
	output.Target("sign-" + image.Name()).
		Description(fmt.Sprintf("Signs the image for %s. Requires interactive Google authentication.", image.ImageName)).
		Script(signImageScriptPrefix + image.ImageName + ":$(IMAGE_TAG)").
		Phony()

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
		if copyFrom.Name == "" {
			return errors.New("copyFrom.name is required")
		}

		output.Stage(copyFrom.Name).From(copyFrom.Stage).Platform(copyFrom.Platform)
		stage.Step(step.Copy(stringOr(copyFrom.Source, "/"), stringOr(copyFrom.Destination, "/")).From(copyFrom.Name))
	}

	if image.meta.GitHubOrganization != "" && image.meta.GitHubRepository != "" {
		stage.Step(step.Label(ImageSourceLabel, fmt.Sprintf("https://github.com/%s/%s", image.meta.GitHubOrganization, image.meta.GitHubRepository)))
	}

	for k, v := range image.Environment {
		stage.Step(step.Env(k, v))
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

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/output/drone"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/common"
	"github.com/siderolabs/kres/internal/project/meta"
)

// ToolchainKind is a Go compiler source.
type ToolchainKind int

// Toolchain kinds.
const (
	ToolchainOfficial = iota
	ToolchainTools
)

// Toolchain provides Go compiler and common utilities.
type Toolchain struct { //nolint:govet
	dag.BaseNode

	meta *meta.Options

	Kind          ToolchainKind
	Version       string
	Image         string
	ExtraPackages []string `yaml:"extraPackages"`
	PrivateRepos  []string `yaml:"privateRepos"`
}

// NewToolchain builds Toolchain with default values.
func NewToolchain(meta *meta.Options) *Toolchain {
	toolchain := &Toolchain{
		BaseNode: dag.NewBaseNode("base"),

		meta: meta,

		Kind:    ToolchainOfficial,
		Version: meta.GoContainerVersion,
	}

	meta.BuildArgs = append(meta.BuildArgs, "TOOLCHAIN", "CGO_ENABLED", "GO_BUILDFLAGS", "GO_LDFLAGS")
	meta.BinPath = toolchain.binPath()
	meta.CachePath = toolchain.cachePath()
	meta.GoPath = "/go"

	return toolchain
}

// AfterLoad adds the github token to the build args in this case, making it possible
// to configure git and go to use private repositories.
func (toolchain *Toolchain) AfterLoad() error {
	if toolchain.PrivateRepos != nil {
		toolchain.meta.BuildArgs = append(toolchain.meta.BuildArgs, "GITHUB_TOKEN")
	}

	return nil
}

func (toolchain *Toolchain) image() string {
	if toolchain.Image != "" {
		return toolchain.Image
	}

	switch toolchain.Kind {
	case ToolchainOfficial:
		return fmt.Sprintf("docker.io/golang:%s", toolchain.Version)
	case ToolchainTools:
		return fmt.Sprintf("ghcr.io/siderolabs/tools:%s", toolchain.Version)
	default:
		panic("unsupported toolchain kind")
	}
}

func (toolchain *Toolchain) binPath() string {
	switch toolchain.Kind {
	case ToolchainOfficial:
		return "/bin"
	case ToolchainTools:
		return "/toolchain/bin"
	default:
		panic("unsupported toolchain kind")
	}
}

func (toolchain *Toolchain) cachePath() string {
	switch toolchain.Kind {
	case ToolchainOfficial:
		return "/root/.cache"
	case ToolchainTools:
		return "/.cache"
	default:
		panic("unsupported toolchain kind")
	}
}

// CompileMakefile implements makefile.Compiler.
func (toolchain *Toolchain) CompileMakefile(output *makefile.Output) error {
	output.VariableGroup(makefile.VariableGroupDocker).
		Variable(makefile.OverridableVariable("TOOLCHAIN", toolchain.image()))

	common := output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable("GO_BUILDFLAGS", "")).
		Variable(makefile.OverridableVariable("GO_LDFLAGS", "")).
		Variable(makefile.OverridableVariable("CGO_ENABLED", "0"))

	// add github token only if necessary
	if toolchain.PrivateRepos != nil {
		common.Variable(makefile.OverridableVariable("GITHUB_TOKEN", ""))
	}

	output.IfTrueCondition("WITH_RACE").
		Then(
			makefile.AppendVariable("GO_BUILDFLAGS", "-race"),
			makefile.SimpleVariable("CGO_ENABLED", "1"),
			makefile.AppendVariable("GO_LDFLAGS", "-linkmode=external -extldflags '-static'"),
		)

	output.IfTrueCondition("WITH_DEBUG").
		Then(
			makefile.AppendVariable("GO_BUILDFLAGS", "-tags sidero.debug"),
		).
		Else(
			makefile.AppendVariable("GO_LDFLAGS", "-s -w"),
		)

	output.Target("base").
		Depends(dag.GatherMatchingInputNames(toolchain, dag.Implements[dockerfile.Generator]())...).
		Description("Prepare base toolchain").
		Script("@$(MAKE) target-$@").
		Phony()

	return nil
}

// CompileDrone implements drone.Compiler.
func (toolchain *Toolchain) CompileDrone(output *drone.Output) error {
	baseStep := drone.MakeStep("base").DependsOn(dag.GatherMatchingInputNames(toolchain, dag.Implements[drone.Compiler]())...)

	if toolchain.PrivateRepos != nil {
		baseStep = baseStep.EnvironmentFromSecret("GITHUB_TOKEN", "github_token")
	}

	output.Step(baseStep)

	return nil
}

// CompileDockerfile implements dockerfile.Compiler.
func (toolchain *Toolchain) CompileDockerfile(output *dockerfile.Output) error {
	output.Arg(step.Arg("TOOLCHAIN"))

	toolchainStage := output.Stage("toolchain").
		Description("base toolchain image").
		From("${TOOLCHAIN}")

	if toolchain.Kind == ToolchainOfficial {
		packages := []string{"add", "bash", "curl", "build-base", "protoc", "protobuf-dev"}
		packages = append(packages, toolchain.ExtraPackages...)

		// automatically add git if we know we're going to have to deal with private repos
		if toolchain.PrivateRepos != nil {
			packages = append(packages, "git")
		}

		toolchainStage.
			Step(step.Run("apk", append([]string{"--update", "--no-cache"}, packages...)...))
	}

	tools := output.Stage("tools").
		Description("build tools").
		From("--platform=${BUILDPLATFORM} toolchain").
		Step(step.Env("GO111MODULE", "on")).
		Step(step.Arg("CGO_ENABLED")).
		Step(step.Env("CGO_ENABLED", "${CGO_ENABLED}")).
		Step(step.Env("GOPATH", toolchain.meta.GoPath))

	// configure git to use the github token for private repos and set GOPRIVATE
	if toolchain.PrivateRepos != nil {
		tools.Step(step.Arg("GITHUB_TOKEN")).
			Step(step.Env("GITHUB_TOKEN", "${GITHUB_TOKEN}")).
			Step(step.Env("GOPRIVATE", strings.Join(toolchain.PrivateRepos, ","))).
			Step(step.Script("git config --global url.https://${GITHUB_TOKEN}:x-oauth-basic@github.com/.insteadOf https://github.com/"))
	}

	if err := dag.WalkNode(toolchain, func(node dag.Node) error {
		if builder, ok := node.(common.ToolchainBuilder); ok {
			return builder.ToolchainBuild(tools)
		}

		return nil
	}, nil, 1); err != nil {
		return err
	}

	base := output.Stage("base").
		Description("tools and sources").
		From("tools").
		Step(step.WorkDir("/src")).
		Step(step.Copy("./go.mod", ".")).
		Step(step.Copy("./go.sum", ".")).
		Step(step.Run("go", "mod", "download").MountCache(filepath.Join(toolchain.meta.GoPath, "pkg"))).
		Step(step.Run("go", "mod", "verify").MountCache(filepath.Join(toolchain.meta.GoPath, "pkg")))

	for _, directory := range toolchain.meta.GoDirectories {
		base.Step(step.Copy("./"+directory, "./"+directory))
	}

	for _, file := range toolchain.meta.GoSourceFiles {
		base.Step(step.Copy("./"+file, "./"+file))
	}

	// build chain of gen containers.
	inputs := dag.GatherMatchingInputs(toolchain, dag.Implements[dockerfile.Generator]())
	for _, input := range inputs {
		for _, path := range input.(dockerfile.Generator).GetArtifacts() { //nolint:forcetypeassert
			base.Step(step.Copy(path, "./"+strings.Trim(path, "/")).From(input.Name()))
		}
	}

	base.Step(step.Script(`go list -mod=readonly all >/dev/null`).MountCache(filepath.Join(toolchain.meta.GoPath, "pkg")))

	return nil
}

// SkipAsMakefileDependency implements makefile.SkipAsMakefileDependency.
func (toolchain *Toolchain) SkipAsMakefileDependency() {
}

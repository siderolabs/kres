// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/talos-systems/kres/internal/dag"
	"github.com/talos-systems/kres/internal/output/dockerfile"
	"github.com/talos-systems/kres/internal/output/dockerfile/step"
	"github.com/talos-systems/kres/internal/output/drone"
	"github.com/talos-systems/kres/internal/output/makefile"
	"github.com/talos-systems/kres/internal/project/common"
	"github.com/talos-systems/kres/internal/project/meta"
)

// ToolchainKind is a Go compiler source.
type ToolchainKind int

// Toolchain kinds.
const (
	ToolchainOfficial = iota
	ToolchainTools
)

// Toolchain provides Go compiler and common utilities.
type Toolchain struct { //nolint: govet
	dag.BaseNode

	meta *meta.Options

	Kind    ToolchainKind
	Version string
	Image   string
}

// NewToolchain builds Toolchain with default values.
func NewToolchain(meta *meta.Options) *Toolchain {
	toolchain := &Toolchain{
		BaseNode: dag.NewBaseNode("base"),

		meta: meta,

		Kind:    ToolchainOfficial,
		Version: "1.16-alpine",
	}

	meta.BuildArgs = append(meta.BuildArgs, "TOOLCHAIN")
	meta.BinPath = toolchain.binPath()
	meta.CachePath = toolchain.cachePath()
	meta.GoPath = "/go"

	return toolchain
}

func (toolchain *Toolchain) image() string {
	if toolchain.Image != "" {
		return toolchain.Image
	}

	switch toolchain.Kind {
	case ToolchainOfficial:
		return fmt.Sprintf("docker.io/golang:%s", toolchain.Version)
	case ToolchainTools:
		return fmt.Sprintf("ghcr.io/talos-systems/tools:%s", toolchain.Version)
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

	output.Target("base").
		Depends(dag.GatherMatchingInputNames(toolchain, dag.Implements((*dockerfile.Generator)(nil)))...).
		Description("Prepare base toolchain").
		Script("@$(MAKE) target-$@").
		Phony()

	return nil
}

// CompileDrone implements drone.Compiler.
func (toolchain *Toolchain) CompileDrone(output *drone.Output) error {
	output.Step(drone.MakeStep("base").
		DependsOn(dag.GatherMatchingInputNames(toolchain, dag.Implements((*drone.Compiler)(nil)))...),
	)

	return nil
}

// CompileDockerfile implements dockerfile.Compiler.
func (toolchain *Toolchain) CompileDockerfile(output *dockerfile.Output) error {
	output.Arg(step.Arg("TOOLCHAIN"))

	toolchainStage := output.Stage("toolchain").
		Description("base toolchain image").
		From("${TOOLCHAIN}")

	if toolchain.Kind == ToolchainOfficial {
		toolchainStage.
			Step(step.Run("apk", "--update", "--no-cache", "add", "bash", "curl", "build-base", "protoc", "protobuf-dev"))
	}

	tools := output.Stage("tools").
		Description("build tools").
		From("toolchain").
		Step(step.Env("GO111MODULE", "on")).
		Step(step.Env("CGO_ENABLED", "0")).
		Step(step.Env("GOPATH", toolchain.meta.GoPath))

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
	inputs := dag.GatherMatchingInputs(toolchain, dag.Implements((*dockerfile.Generator)(nil)))
	for _, input := range inputs {
		for _, path := range input.(dockerfile.Generator).GetArtifacts() {
			base.Step(step.Copy(path, "./"+strings.Trim(path, "/")).From(input.Name()))
		}
	}

	base.Step(step.Script(`go list -mod=readonly all >/dev/null`).MountCache(filepath.Join(toolchain.meta.GoPath, "pkg")))

	return nil
}

// SkipAsMakefileDependency implements makefile.SkipAsMakefileDependency.
func (toolchain *Toolchain) SkipAsMakefileDependency() {
}

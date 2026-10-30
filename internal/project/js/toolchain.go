// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package js

import (
	"path/filepath"
	"strings"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/output/gitignore"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/output/template"
	"github.com/siderolabs/kres/internal/project/common"
	"github.com/siderolabs/kres/internal/project/js/templates"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Toolchain provides node js runtime and common utilities.
type Toolchain struct {
	dag.BaseNode

	meta *meta.Options

	sourceDir     string
	Version       string
	Image         string
	ExtraPackages []string `yaml:"extraPackages"`
}

// NewToolchain builds Toolchain with default values.
func NewToolchain(meta *meta.Options, sourceDir string) *Toolchain {
	toolchain := &Toolchain{
		BaseNode: dag.NewBaseNode("js"),

		meta:      meta,
		sourceDir: sourceDir,

		Version: config.NodeContainerImageVersion,
	}

	meta.BuildArgs = append(meta.BuildArgs, "JS_TOOLCHAIN")

	return toolchain
}

// CompileGitignore implements gitignore.Compiler.
func (toolchain *Toolchain) CompileGitignore(output *gitignore.Output) error {
	output.
		IgnorePath(filepath.Join(toolchain.sourceDir, "node_modules"))

	return nil
}

// CompileTemplates implements template.Compiler.
func (toolchain *Toolchain) CompileTemplates(output *template.Output) error {
	output.Define(filepath.Join(toolchain.sourceDir, "tsconfig.json"), templates.TSConfig).
		NoPreamble().
		NoOverwrite()

	return nil
}

func (toolchain *Toolchain) image() string {
	if toolchain.Image != "" {
		return toolchain.Image
	}

	return "docker.io/node:" + toolchain.Version
}

// CompileMakefile implements makefile.Compiler.
func (toolchain *Toolchain) CompileMakefile(output *makefile.Output) error {
	output.VariableGroup(makefile.VariableGroupDocker).
		Variable(makefile.OverridableVariable("JS_TOOLCHAIN", toolchain.image()))

	output.Target("js").
		Description("Prepare js base toolchain.").
		Script("@$(MAKE) target-$@").
		Phony()

	return nil
}

// CompileGitHubWorkflow implements ghworkflow.Compiler.
func (toolchain *Toolchain) CompileGitHubWorkflow(output *ghworkflow.Output) error {
	output.AddStep(ghworkflow.DefaultJobName, ghworkflow.Step("js").SetMakeStep("js"))

	return nil
}

// packages returns the system packages to install in the toolchain image: those the enabled features need,
// followed by the extra packages from the configuration.
func (toolchain *Toolchain) packages() ([]string, error) {
	var packages []string

	if err := dag.WalkNode(toolchain, func(node dag.Node) error {
		if packager, ok := node.(common.ToolchainPackager); ok {
			packages = append(packages, packager.ToolchainPackages()...)
		}

		return nil
	}, nil, 1); err != nil {
		return nil, err
	}

	packages = append(packages, toolchain.ExtraPackages...)

	seen := make(map[string]struct{}, len(packages))
	unique := make([]string, 0, len(packages))

	for _, pkg := range packages {
		if _, ok := seen[pkg]; ok {
			continue
		}

		seen[pkg] = struct{}{}
		unique = append(unique, pkg)
	}

	return unique, nil
}

// CompileDockerfile implements dockerfile.Compiler.
func (toolchain *Toolchain) CompileDockerfile(output *dockerfile.Output) error {
	output.Arg(step.Arg("JS_TOOLCHAIN"))

	toolchain.meta.JSCachePath = "/root/.npm"

	packages, err := toolchain.packages()
	if err != nil {
		return err
	}

	toolchainStage := output.Stage("js-toolchain").
		Description("base toolchain image").
		From("--platform=${BUILDPLATFORM} ${JS_TOOLCHAIN}")

	// Without packages to install, the toolchain image may be any image with Node.js, not only an Alpine one.
	if len(packages) > 0 {
		toolchainStage.Step(step.Run("apk", append([]string{"--update", "--no-cache", "add"}, packages...)...))
	}

	toolchainStage.
		Step(step.Copy("./go.mod", ".")).
		Step(step.Copy("./go.sum", ".")).
		Step(step.Env("GOPATH", toolchain.meta.GoPath)).
		Step(step.Env("PATH", "${PATH}:/usr/local/go/bin"))

	base := output.Stage("js").
		Description("tools and sources").
		From("--platform=${BUILDPLATFORM} js-toolchain").
		Step(step.WorkDir("/src"))

	if err := dag.WalkNode(toolchain, func(node dag.Node) error {
		if builder, ok := node.(common.ToolchainBuilder); ok {
			return builder.ToolchainBuild(base)
		}

		return nil
	}, nil, 1); err != nil {
		return err
	}

	// Install dependencies from the manifests alone, so that the npm ci layer stays cached when only sources change.
	base.Step(step.Copy(filepath.Join(toolchain.sourceDir, "package*.json"), "./")).
		Step(step.Copy(filepath.Join(toolchain.sourceDir, ".npmrc"), "./")).
		Step(step.Script("npm ci").
			MountCache(toolchain.meta.JSCachePath, toolchain.meta.GitHubRepository, step.CacheLocked))

	base.Step(step.Copy(filepath.Join(toolchain.sourceDir, "*.json"), "./")).
		Step(step.Copy(filepath.Join(toolchain.sourceDir, "*.js"), "./")).
		Step(step.Copy(filepath.Join(toolchain.sourceDir, "*.ts"), "./")).
		Step(step.Copy(filepath.Join(toolchain.sourceDir, "*.html"), "./")).
		Step(step.Copy(filepath.Join(toolchain.sourceDir, ".editorconfig"), "./")).
		Step(step.Copy(filepath.Join(toolchain.sourceDir, ".gitignore"), "./")).
		Step(step.Copy(filepath.Join(toolchain.sourceDir, ".prettier*"), "./"))

	for _, directory := range toolchain.meta.JSDirectories {
		dest := strings.TrimPrefix(directory, toolchain.sourceDir)

		base.Step(step.Copy("./"+directory, "./"+strings.Trim(dest, "/")))
	}

	return nil
}

// SkipAsMakefileDependency implements makefile.SkipAsMakefileDependency.
func (toolchain *Toolchain) SkipAsMakefileDependency() {
}

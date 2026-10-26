// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package js

import (
	"path"
	"path/filepath"

	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/output/lefthook"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Generate generates frontend sources: the compiled protobuf specs, and the outputs of an npm script.
//
// The outputs are copied back into the source tree by `make generate-<name>`, which `make generate` runs.
type Generate struct {
	dag.BaseNode

	meta     *meta.Options
	protobuf *Protobuf

	// Script is the npm script that generates sources, run in the frontend directory.
	Script string `yaml:"script"`
	// ScriptInputs are the paths, relative to the repository root, that the script reads from outside the
	// frontend directory. Each is available at the same path relative to the frontend directory's parent,
	// so that `../api/openapi.yaml` in the script resolves to the repository's `api/openapi.yaml`.
	ScriptInputs []string `yaml:"inputs"`
	// ScriptOutputs are the paths, relative to the frontend directory, that the script generates.
	ScriptOutputs []string `yaml:"outputs"`
}

// NewGenerate builds Generate node.
func NewGenerate(meta *meta.Options, name string, protobuf *Protobuf) *Generate {
	return &Generate{
		BaseNode: dag.NewBaseNode(name),

		meta:     meta,
		protobuf: protobuf,
	}
}

func (generate *Generate) enabled() bool {
	return generate.protobuf.enabled() || generate.Script != ""
}

func (generate *Generate) targetName() string {
	return "generate-" + generate.Name()
}

// CompileMakefile implements makefile.Compiler.
func (generate *Generate) CompileMakefile(output *makefile.Output) error {
	if !generate.enabled() {
		return nil
	}

	output.Target(generate.targetName()).
		Description("Generate frontend sources.").
		Script("@$(MAKE) local-$@ DEST=./")

	output.OptionalDepends("generate", generate.targetName())

	return nil
}

// CompileDockerfile implements dockerfile.Compiler.
func (generate *Generate) CompileDockerfile(output *dockerfile.Output) error {
	stage := output.Stage(generate.targetName()).
		Description("cleaned up specs and compiled versions").
		From("scratch")

	for _, file := range generate.protobuf.Files {
		stage.Step(step.Add(file.Source, file.Destination))
	}

	if generate.protobuf.enabled() {
		stage.Step(step.Copy(filepath.Clean(generate.Name())+"/", filepath.Clean(generate.Name())+"/").
			From(generate.protobuf.compileContainer()))
	}

	if generate.Script == "" {
		return nil
	}

	// The js stage holds the frontend directory at /src, so the repository root maps to /.
	runContainer := "npm-generate-" + generate.Name()
	run := output.Stage(runContainer).
		Description("runs the npm generate script").
		From("js")

	for _, input := range generate.ScriptInputs {
		run.Step(step.Copy(input, path.Join("/", input)))
	}

	run.Step(step.Script("npm run " + generate.Script))

	for _, out := range generate.ScriptOutputs {
		stage.Step(step.Copy(path.Join("/src", out), path.Join(generate.Name(), out)).From(runContainer))
	}

	return nil
}

// CompileLefthook implements lefthook.Compiler.
func (generate *Generate) CompileLefthook(output *lefthook.Output) error {
	if !generate.enabled() {
		return nil
	}

	output.Hook(lefthook.HookGroupPreCommit).
		Group(lefthook.PreCommitFixStage).
		WithParallel(false).
		Job().
		WithName("generate "+generate.Name()).
		WithRun("make "+generate.targetName()).
		WithEnv("USERNAME", generate.meta.GitHubOrganization).
		WithStageFixed()

	return nil
}

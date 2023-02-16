// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package dockerfile implements output to Dockerfiles.
package dockerfile

import (
	"fmt"
	"io"
	"sort"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/output"
	"github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/toposort"
)

const (
	dockerfile   = "Dockerfile"
	dockerignore = ".dockerignore"
	syntax       = "docker/dockerfile-upstream:" + config.DockerfileFrontendImageVersion
)

// Output implements Dockerfile and .dockerignore generation.
type Output struct {
	output.FileAdapter

	args   []*step.ArgStep
	stages map[string]*Stage

	allowedLocalPaths []string
}

// NewOutput creates new dockerfile output.
func NewOutput() *Output {
	output := &Output{}

	output.FileAdapter.FileWriter = output

	return output
}

// Compile implements [output.TypedWriter] interface.
func (o *Output) Compile(compiler Compiler) error {
	return compiler.CompileDockerfile(o)
}

// Filenames implements output.FileWriter interface.
func (o *Output) Filenames() []string {
	return []string{dockerfile, dockerignore}
}

// GenerateFile implements output.FileWriter interface.
func (o *Output) GenerateFile(filename string, w io.Writer) error {
	switch filename {
	case dockerfile:
		return o.dockerfile(w)
	case dockerignore:
		return o.dockerignore(w)
	default:
		panic("unexpected filename: " + filename)
	}
}

// Stage creates new stage.
func (o *Output) Stage(name string) *Stage {
	stage := &Stage{name: name}

	if o.stages == nil {
		o.stages = make(map[string]*Stage)
	}

	o.stages[name] = stage

	return stage
}

// Arg appends new arg.
func (o *Output) Arg(arg *step.ArgStep) *Output {
	o.args = append(o.args, arg)

	return o
}

// AllowLocalPath adds path to the list of paths to be copied into the context.
func (o *Output) AllowLocalPath(paths ...string) *Output {
	o.allowedLocalPaths = append(o.allowedLocalPaths, paths...)

	return o
}

func (o *Output) dockerfile(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "# syntax = %s\n\n", syntax); err != nil {
		return err
	}

	if _, err := w.Write([]byte(output.Preamble("# "))); err != nil {
		return err
	}

	for _, arg := range o.args {
		if err := arg.Generate(w); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	stageNodes := make([]*Stage, 0, len(o.stages))
	for _, stage := range o.stages {
		stageNodes = append(stageNodes, stage)
	}

	sort.Slice(stageNodes, func(i, j int) bool {
		return stageNodes[i].name < stageNodes[j].name
	})

	sortedStages, _ := toposort.Stable(stageNodes)

	for _, stageNode := range sortedStages {
		if err := stageNode.Generate(w); err != nil {
			return err
		}
	}

	return nil
}

func (o *Output) dockerignore(w io.Writer) error {
	if _, err := w.Write([]byte(output.Preamble("# "))); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, "*"); err != nil {
		return err
	}

	for _, path := range o.allowedLocalPaths {
		if _, err := fmt.Fprintf(w, "!%s\n", path); err != nil {
			return err
		}
	}

	return nil
}

// Compiler is implemented by project blocks which support Dockerfile generate.
type Compiler interface {
	CompileDockerfile(*Output) error
}

// Generator is implemented by project blocks which generate code.
type Generator interface {
	GetArtifacts() []string
}

// CmdCompiler is implemented by project blocks which may output executable entrypoints.
type CmdCompiler interface {
	Entrypoint() string
}

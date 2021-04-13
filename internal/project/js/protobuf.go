// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package js

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/talos-systems/kres/internal/dag"
	"github.com/talos-systems/kres/internal/output/dockerfile"
	"github.com/talos-systems/kres/internal/output/dockerfile/step"
	"github.com/talos-systems/kres/internal/output/makefile"
	"github.com/talos-systems/kres/internal/project/meta"
)

// Protobuf provides .proto compilation with ts-proto plugin.
type Protobuf struct {
	dag.BaseNode

	meta *meta.Options

	ProtobufTSVersion string `yaml:"protobufTSVersion"`

	BaseSpecPath string `yaml:"baseSpecPath"`

	Specs []ProtoSpec `yaml:"specs"`

	ExperimentalFlags []string `yaml:"experimentalFlags"`
}

// ProtoSpec describes a set of protobuf specs to be compiled.
type ProtoSpec struct {
	Source          string `yaml:"source"`
	SubDirectory    string `yaml:"subdirectory"`
	DestinationRoot string `yaml:"destinationRoot"`
}

// NewProtobuf builds Protobuf node.
func NewProtobuf(meta *meta.Options, name string) *Protobuf {
	meta.BuildArgs = append(meta.BuildArgs,
		"PROTOBUF_TS_VERSION",
	)

	return &Protobuf{
		BaseNode: dag.NewBaseNode(name),

		meta: meta,

		ProtobufTSVersion: "v1.79.2",

		BaseSpecPath: "/api",
	}
}

// CompileMakefile implements makefile.Compiler.
func (proto *Protobuf) CompileMakefile(output *makefile.Output) error {
	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable("PROTOBUF_TS_VERSION", strings.TrimLeft(proto.ProtobufTSVersion, "v")))

	if len(proto.Specs) == 0 {
		return nil
	}

	output.Target(fmt.Sprintf("generate-%s", proto.Name())).Description("Generate .proto definitions.").
		Script("@$(MAKE) local-$@ DEST=./")

	return nil
}

// ToolchainBuild implements common.ToolchainBuilder hook.
func (proto *Protobuf) ToolchainBuild(stage *dockerfile.Stage) error {
	if len(proto.Specs) == 0 {
		return nil
	}

	stage.
		Step(step.Arg("PROTOBUF_TS_VERSION")).
		Step(step.Script("npm install -g ts-proto@^${PROTOBUF_TS_VERSION}"))

	return nil
}

// CompileDockerfile implements dockerfile.Compiler.
func (proto *Protobuf) CompileDockerfile(output *dockerfile.Output) error {
	rootDir := "/" + proto.Name()
	generateContainer := fmt.Sprintf("generate-%s", proto.Name())
	specsContainer := fmt.Sprintf("proto-specs-%s", proto.Name())
	compileContainer := fmt.Sprintf("proto-compile-%s", proto.Name())

	generate := output.Stage(generateContainer).
		Description("cleaned up specs and compiled versions").
		From("scratch")

	if len(proto.Specs) == 0 {
		return nil
	}

	specs := output.Stage(specsContainer).
		Description("collects proto specs").
		From("scratch")

	for _, spec := range proto.Specs {
		specs.Step(
			step.Add(spec.Source, filepath.Join(rootDir, spec.DestinationRoot, spec.SubDirectory)+"/"),
		)
	}

	compile := output.Stage(compileContainer).
		Description("runs protobuf compiler").
		From("js").
		Step(step.Copy("/", "/").From(specsContainer))

	for _, spec := range proto.Specs {
		dir := filepath.Join(rootDir, spec.DestinationRoot)
		compile.Step(
			step.Run(
				"protoc",
				append(
					append([]string{
						fmt.Sprintf("-I%s", dir),
						fmt.Sprintf("--ts_proto_out=paths=source_relative:%s", dir),
						"--plugin=/root/.npm-global/.bin/protoc-gen-ts_proto.cmd",
					},
						proto.ExperimentalFlags...,
					),
					filepath.Join(dir, spec.SubDirectory, filepath.Base(spec.Source)),
				)...,
			),
		)

		if !strings.HasPrefix(spec.Source, "http") {
			compile.Step(step.Script(fmt.Sprintf("find %s -name \"*.proto\" | xargs rm", dir)))
		}
	}

	generate.Step(step.Copy(filepath.Clean(proto.Name())+"/", filepath.Clean(proto.Name())+"/").
		From(compileContainer))

	return nil
}

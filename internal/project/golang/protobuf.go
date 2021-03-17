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
	"github.com/talos-systems/kres/internal/output/makefile"
	"github.com/talos-systems/kres/internal/project/meta"
)

// Protobuf provides .proto compilation with grpc-go plugin.
type Protobuf struct {
	dag.BaseNode

	meta *meta.Options

	ProtobufGoVersion string `yaml:"protobufGoVersion"`
	GrpcGoVersion     string `yaml:"grpcGoVersion"`

	BaseSpecPath string `yaml:"baseSpecPath"`

	Specs []ProtoSpec `yaml:"specs"`

	ExperimentalFlags []string `yaml:"experimentalFlags"`
}

// ProtoSpec describes a set of protobuf specs to be compiled.
type ProtoSpec struct {
	Source       string `yaml:"source"`
	SubDirectory string `yaml:"subdirectory"`
}

// NewProtobuf builds Protobuf node.
func NewProtobuf(meta *meta.Options) *Protobuf {
	meta.BuildArgs = append(meta.BuildArgs,
		"PROTOBUF_GO_VERSION",
		"GRPC_GO_VERSION",
	)

	return &Protobuf{
		BaseNode: dag.NewBaseNode("protobuf"),

		meta: meta,

		ProtobufGoVersion: "v1.25.0",
		GrpcGoVersion:     "v1.1.0",

		BaseSpecPath: "/api",
	}
}

// CompileMakefile implements makefile.Compiler.
func (proto *Protobuf) CompileMakefile(output *makefile.Output) error {
	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable("PROTOBUF_GO_VERSION", strings.TrimLeft(proto.ProtobufGoVersion, "v"))).
		Variable(makefile.OverridableVariable("GRPC_GO_VERSION", strings.TrimLeft(proto.GrpcGoVersion, "v")))

	if len(proto.Specs) == 0 {
		return nil
	}

	output.Target("generate").Description("Generate .proto definitions.").
		Script("@$(MAKE) local-$@ DEST=./")

	return nil
}

// ToolchainBuild implements common.ToolchainBuilder hook.
func (proto *Protobuf) ToolchainBuild(stage *dockerfile.Stage) error {
	if len(proto.Specs) == 0 {
		return nil
	}

	stage.
		Step(step.Arg("PROTOBUF_GO_VERSION")).
		Step(step.Script("go install google.golang.org/protobuf/cmd/protoc-gen-go@v${PROTOBUF_GO_VERSION}")).
		Step(step.Run("mv", filepath.Join(proto.meta.GoPath, "bin", "protoc-gen-go"), proto.meta.BinPath)).
		Step(step.Arg("GRPC_GO_VERSION")).
		Step(step.Script("go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v${GRPC_GO_VERSION}")).
		Step(step.Run("mv", filepath.Join(proto.meta.GoPath, "bin", "protoc-gen-go-grpc"), proto.meta.BinPath))

	return nil
}

// CompileDockerfile implements dockerfile.Compiler.
func (proto *Protobuf) CompileDockerfile(output *dockerfile.Output) error {
	generate := output.Stage("generate").
		Description("cleaned up specs and compiled versions").
		From("scratch")

	if len(proto.Specs) == 0 {
		return nil
	}

	specs := output.Stage("proto-specs").
		Description("collects proto specs").
		From("scratch")

	for _, spec := range proto.Specs {
		specs.Step(
			step.Add(spec.Source, filepath.Join(proto.BaseSpecPath, spec.SubDirectory)+"/"),
		)
	}

	compile := output.Stage("proto-compile").
		Description("runs protobuf compiler").
		From("tools").
		Step(step.Copy("/", "/").From("proto-specs"))

	for _, spec := range proto.Specs {
		compile.Step(
			step.Run(
				"protoc",
				append(
					append([]string{
						fmt.Sprintf("-I%s", proto.BaseSpecPath),
						fmt.Sprintf("--go_out=paths=source_relative:%s", proto.BaseSpecPath),
						fmt.Sprintf("--go-grpc_out=paths=source_relative:%s", proto.BaseSpecPath),
					},
						proto.ExperimentalFlags...,
					),
					filepath.Join(proto.BaseSpecPath, spec.SubDirectory, filepath.Base(spec.Source)),
				)...,
			),
		)
	}

	generate.Step(step.Copy(filepath.Clean(proto.BaseSpecPath)+"/", filepath.Clean(proto.BaseSpecPath)+"/").
		From("proto-compile"))

	return nil
}

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

	ProtobufGoVersion  string `yaml:"protobufGoVersion"`
	GrpcGoVersion      string `yaml:"grpcGoVersion"`
	GrpcGatewayVersion string `yaml:"grpcGatewayVersion"`
	VTProtobufVersion  string `yaml:"vtProtobufVersion"`

	BaseSpecPath string `yaml:"baseSpecPath"`

	Specs []ProtoSpec `yaml:"specs"`

	ExperimentalFlags []string `yaml:"experimentalFlags"`

	VTProtobufEnabled bool `yaml:"vtProtobufEnabled"`
}

// ProtoSpec describes a set of protobuf specs to be compiled.
type ProtoSpec struct {
	Source       string `yaml:"source"`
	SubDirectory string `yaml:"subdirectory"`
	SkipCompile  bool   `yaml:"skipCompile"`
	GenGateway   bool   `yaml:"genGateway"`
}

// NewProtobuf builds Protobuf node.
func NewProtobuf(meta *meta.Options) *Protobuf {
	meta.BuildArgs = append(meta.BuildArgs,
		"PROTOBUF_GO_VERSION",
		"GRPC_GO_VERSION",
		"GRPC_GATEWAY_VERSION",
		"VTPROTOBUF_VERSION",
	)

	return &Protobuf{
		BaseNode: dag.NewBaseNode("protobuf"),

		meta: meta,

		ProtobufGoVersion:  "v1.27.1",
		GrpcGoVersion:      "v1.1.0",
		GrpcGatewayVersion: "v2.4.0",
		VTProtobufVersion:  "81d623a9a700ede8ef765e5ab08b3aa1f5b4d5a8",

		BaseSpecPath: "/api",
	}
}

// CompileMakefile implements makefile.Compiler.
func (proto *Protobuf) CompileMakefile(output *makefile.Output) error {
	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable("PROTOBUF_GO_VERSION", strings.TrimLeft(proto.ProtobufGoVersion, "v"))).
		Variable(makefile.OverridableVariable("GRPC_GO_VERSION", strings.TrimLeft(proto.GrpcGoVersion, "v"))).
		Variable(makefile.OverridableVariable("GRPC_GATEWAY_VERSION", strings.TrimLeft(proto.GrpcGatewayVersion, "v"))).
		Variable(makefile.OverridableVariable("VTPROTOBUF_VERSION", strings.TrimLeft(proto.VTProtobufVersion, "v")))

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
		Step(step.Run("mv", filepath.Join(proto.meta.GoPath, "bin", "protoc-gen-go-grpc"), proto.meta.BinPath)).
		Step(step.Arg("GRPC_GATEWAY_VERSION")).
		Step(step.Script("go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v${GRPC_GATEWAY_VERSION}")).
		Step(step.Run("mv", filepath.Join(proto.meta.GoPath, "bin", "protoc-gen-grpc-gateway"), proto.meta.BinPath))

	if proto.VTProtobufEnabled {
		stage.
			Step(step.Arg("VTPROTOBUF_VERSION")).
			Step(step.Script("go install github.com/planetscale/vtprotobuf/cmd/protoc-gen-go-vtproto@${VTPROTOBUF_VERSION}")).
			Step(step.Run("mv", filepath.Join(proto.meta.GoPath, "bin", "protoc-gen-go-vtproto"), proto.meta.BinPath))
	}

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

	cleanupSteps := []*step.RunStep{}

	for _, spec := range proto.Specs {
		external := strings.HasPrefix(spec.Source, "http")
		source := filepath.Join(proto.BaseSpecPath, spec.SubDirectory, filepath.Base(spec.Source))

		if !spec.SkipCompile {
			flags := []string{
				fmt.Sprintf("-I%s", proto.BaseSpecPath),
			}

			if spec.GenGateway {
				flags = append(flags,
					fmt.Sprintf("--grpc-gateway_out=paths=source_relative:%s", proto.BaseSpecPath),
					"--grpc-gateway_opt=generate_unbound_methods=true",
				)

				if external {
					flags = append(flags,
						"--grpc-gateway_opt=standalone=true",
					)
				}
			}

			if !spec.GenGateway || !external {
				flags = append(flags,
					fmt.Sprintf("--go_out=paths=source_relative:%s", proto.BaseSpecPath),
					fmt.Sprintf("--go-grpc_out=paths=source_relative:%s", proto.BaseSpecPath),
				)

				if proto.VTProtobufEnabled {
					flags = append(flags,
						fmt.Sprintf("--go-vtproto_out=paths=source_relative:%s", proto.BaseSpecPath),
						"--go-vtproto_opt=features=marshal+unmarshal+size",
					)
				}
			}

			flags = append(flags, proto.ExperimentalFlags...)
			flags = append(flags, source)

			compile.Step(
				step.Run(
					"protoc",
					flags...,
				),
			)
		}

		if !external {
			cleanupSteps = append(cleanupSteps,
				step.Script(fmt.Sprintf("rm %s", source)),
			)
		}
	}

	for _, s := range cleanupSteps {
		compile.Step(s)
	}

	generate.Step(step.Copy(filepath.Clean(proto.BaseSpecPath)+"/", filepath.Clean(proto.BaseSpecPath)+"/").
		From("proto-compile"))

	return nil
}

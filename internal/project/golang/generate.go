// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/output/license"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Generate provides .proto compilation with grpc-go plugin
// and go generate runner.
type Generate struct {
	dag.BaseNode

	meta *meta.Options

	ProtobufGoVersion  string `yaml:"protobufGoVersion"`
	GrpcGoVersion      string `yaml:"grpcGoVersion"`
	GrpcGatewayVersion string `yaml:"grpcGatewayVersion"`
	VTProtobufVersion  string `yaml:"vtProtobufVersion"`

	BaseSpecPath string `yaml:"baseSpecPath"`

	Specs           []ProtoSpec      `yaml:"specs"`
	GoGenerateSpecs []GoGenerateSpec `yaml:"goGenerateSpecs"`

	ExperimentalFlags []string `yaml:"experimentalFlags"`

	VTProtobufEnabled bool `yaml:"vtProtobufEnabled"`
}

// ProtoSpec describes a set of protobuf specs to be compiled.
type ProtoSpec struct {
	Source       string `yaml:"source"`
	External     *bool  `yaml:"external"`
	SubDirectory string `yaml:"subdirectory"`

	sourcePath string

	SkipCompile bool `yaml:"skipCompile"`
	GenGateway  bool `yaml:"genGateway"`

	external bool
}

// GoGenerateSpec describes a set of go generate specs to be compiled.
type GoGenerateSpec struct {
	Source string   `yaml:"source"`
	Copy   []string `yaml:"copy"`
}

// NewGenerate builds Generate node.
func NewGenerate(meta *meta.Options) *Generate {
	meta.BuildArgs = append(meta.BuildArgs,
		"PROTOBUF_GO_VERSION",
		"GRPC_GO_VERSION",
		"GRPC_GATEWAY_VERSION",
		"VTPROTOBUF_VERSION",
	)

	return &Generate{
		BaseNode: dag.NewBaseNode("protobuf"),

		meta: meta,

		ProtobufGoVersion:  config.ProtobufGoVersion,
		GrpcGoVersion:      config.GrpcGoVersion,
		GrpcGatewayVersion: config.GrpcGatewayVersion,
		VTProtobufVersion:  config.VTProtobufVersion,

		BaseSpecPath: "/api",
	}
}

// CompileMakefile implements makefile.Compiler.
func (generate *Generate) CompileMakefile(output *makefile.Output) error {
	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable("PROTOBUF_GO_VERSION", strings.TrimLeft(generate.ProtobufGoVersion, "v"))).
		Variable(makefile.OverridableVariable("GRPC_GO_VERSION", strings.TrimLeft(generate.GrpcGoVersion, "v"))).
		Variable(makefile.OverridableVariable("GRPC_GATEWAY_VERSION", strings.TrimLeft(generate.GrpcGatewayVersion, "v"))).
		Variable(makefile.OverridableVariable("VTPROTOBUF_VERSION", strings.TrimLeft(generate.VTProtobufVersion, "v")))

	if len(generate.Specs) == 0 && len(generate.GoGenerateSpecs) == 0 {
		return nil
	}

	output.Target("generate").Description("Generate .proto definitions.").
		Script("@$(MAKE) local-$@ DEST=./")

	return nil
}

// ToolchainBuild implements common.ToolchainBuilder hook.
func (generate *Generate) ToolchainBuild(stage *dockerfile.Stage) error {
	if len(generate.Specs) == 0 {
		return nil
	}

	stage.
		Step(step.Arg("PROTOBUF_GO_VERSION")).
		Step(step.Script("go install google.golang.org/protobuf/cmd/protoc-gen-go@v${PROTOBUF_GO_VERSION}")).
		Step(step.Run("mv", filepath.Join(generate.meta.GoPath, "bin", "protoc-gen-go"), generate.meta.BinPath)).
		Step(step.Arg("GRPC_GO_VERSION")).
		Step(step.Script("go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v${GRPC_GO_VERSION}")).
		Step(step.Run("mv", filepath.Join(generate.meta.GoPath, "bin", "protoc-gen-go-grpc"), generate.meta.BinPath)).
		Step(step.Arg("GRPC_GATEWAY_VERSION")).
		Step(step.Script("go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v${GRPC_GATEWAY_VERSION}")).
		Step(step.Run("mv", filepath.Join(generate.meta.GoPath, "bin", "protoc-gen-grpc-gateway"), generate.meta.BinPath))

	if generate.VTProtobufEnabled {
		stage.
			Step(step.Arg("VTPROTOBUF_VERSION")).
			Step(step.Script("go install github.com/planetscale/vtprotobuf/cmd/protoc-gen-go-vtproto@v${VTPROTOBUF_VERSION}")).
			Step(step.Run("mv", filepath.Join(generate.meta.GoPath, "bin", "protoc-gen-go-vtproto"), generate.meta.BinPath))
	}

	return nil
}

// CompileDockerfile implements dockerfile.Compiler.
//
//nolint:gocognit,gocyclo,cyclop
func (generate *Generate) CompileDockerfile(output *dockerfile.Output) error {
	generateStage := output.Stage("generate").
		Description("cleaned up specs and compiled versions").
		From("scratch")

	if len(generate.Specs) > 0 {
		specs := output.Stage("proto-specs").
			Description("collects proto specs").
			From("scratch")

		for _, spec := range generate.Specs {
			specs.Step(
				step.Add(spec.Source, filepath.Join(generate.BaseSpecPath, spec.SubDirectory)+"/"),
			)
		}

		for i := range generate.Specs {
			if generate.Specs[i].External != nil {
				generate.Specs[i].external = *generate.Specs[i].External
			} else if strings.HasPrefix(generate.Specs[i].Source, "http") {
				generate.Specs[i].external = true
			}

			generate.Specs[i].sourcePath = filepath.Join(generate.BaseSpecPath, generate.Specs[i].SubDirectory, filepath.Base(generate.Specs[i].Source))
		}

		compile := output.Stage("proto-compile").
			Description("runs protobuf compiler").
			From("tools").
			Step(step.Copy("/", "/").From("proto-specs"))

		var (
			prevFlags              []string
			accumulatedSourcePaths []string
		)

		// try to combine as many specs as possible into a single invocation of protoc,
		// as for some generators this fixes the problem with multiple definitions of internal functions
		for _, spec := range generate.Specs {
			if spec.SkipCompile {
				continue
			}

			flags := []string{
				fmt.Sprintf("-I%s", generate.BaseSpecPath),
			}

			if spec.GenGateway {
				flags = append(flags,
					fmt.Sprintf("--grpc-gateway_out=paths=source_relative:%s", generate.BaseSpecPath),
					"--grpc-gateway_opt=generate_unbound_methods=true",
				)

				if spec.external {
					flags = append(flags,
						"--grpc-gateway_opt=standalone=true",
					)
				}
			}

			if !spec.GenGateway || !spec.external {
				flags = append(flags,
					fmt.Sprintf("--go_out=paths=source_relative:%s", generate.BaseSpecPath),
					fmt.Sprintf("--go-grpc_out=paths=source_relative:%s", generate.BaseSpecPath),
				)

				if generate.VTProtobufEnabled {
					flags = append(flags,
						fmt.Sprintf("--go-vtproto_out=paths=source_relative:%s", generate.BaseSpecPath),
						"--go-vtproto_opt=features=marshal+unmarshal+size",
					)
				}
			}

			flags = append(flags, generate.ExperimentalFlags...)

			if prevFlags != nil && !reflect.DeepEqual(flags, prevFlags) {
				compile.Step(
					step.Run(
						"protoc",
						append(prevFlags, accumulatedSourcePaths...)...,
					),
				)

				accumulatedSourcePaths = nil
			}

			prevFlags = flags

			accumulatedSourcePaths = append(accumulatedSourcePaths, spec.sourcePath)
		}

		if len(accumulatedSourcePaths) > 0 {
			compile.Step(
				step.Run(
					"protoc",
					append(prevFlags, accumulatedSourcePaths...)...,
				),
			)
		}

		// cleanup copied source files
		for _, spec := range generate.Specs {
			if spec.external {
				continue
			}

			compile.Step(
				step.Run(
					"rm",
					spec.sourcePath,
				),
			)
		}

		// gofumpt + goimports
		compile.Step(
			step.Run(
				"goimports",
				"-w",
				"-local",
				generate.meta.CanonicalPath,
				generate.BaseSpecPath,
			),
		)

		compile.Step(
			step.Run(
				"gofumpt",
				"-w",
				generate.BaseSpecPath,
			),
		)

		generateStage.Step(step.Copy(filepath.Clean(generate.BaseSpecPath)+"/", filepath.Clean(generate.BaseSpecPath)+"/").
			From("proto-compile"))
	}

	if len(generate.GoGenerateSpecs) > 0 {
		output.AllowLocalPath(license.Header)
	}

	for index, spec := range generate.GoGenerateSpecs {
		output.Stage(fmt.Sprintf("go-generate-%d", index)).
			Description("run go generate").
			From("base").
			Step(step.WorkDir("/src")).
			Step(step.Copy(license.Header, filepath.Join("./hack/", license.Header))).
			Step(step.Script(fmt.Sprintf("go generate %s/...", spec.Source)).
				MountCache(filepath.Join(generate.meta.CachePath, "go-build")).
				MountCache(filepath.Join(generate.meta.GoPath, "pkg")),
			).
			Step(step.Run(
				"goimports",
				"-w",
				"-local",
				generate.meta.CanonicalPath,
				spec.Source,
			))
	}

	for index, spec := range generate.GoGenerateSpecs {
		for _, path := range spec.Copy {
			path = filepath.Clean(path)
			generateStage.Step(step.Copy(filepath.Join("/src", path), path).From(fmt.Sprintf("go-generate-%d", index)))
		}
	}

	return nil
}

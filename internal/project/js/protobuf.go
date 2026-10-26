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
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Protobuf provides .proto compilation with ts-proto plugin.
type Protobuf struct {
	dag.BaseNode

	meta *meta.Options

	// Files are the arbitrary files to be copied into the image.
	Files []File `yaml:"files"`

	ProtobufTSGatewayVersion string `yaml:"protobufTSGatewayVersion"`

	BaseSpecPath    string `yaml:"baseSpecPath"`
	DestinationRoot string `yaml:"destinationRoot"`

	Specs []ProtoSpec `yaml:"specs"`

	ExperimentalFlags []string `yaml:"experimentalFlags"`
}

// File represents a file to be fetched/copied into the image.
type File struct {
	Source      string `yaml:"source"`
	Destination string `yaml:"destination"`
}

// ProtoSpec describes a set of protobuf specs to be compiled.
type ProtoSpec struct {
	Source          string `yaml:"source"`
	SubDirectory    string `yaml:"subdirectory"`
	DestinationRoot string `yaml:"destinationRoot"`
}

// NewProtobuf builds Protobuf node.
func NewProtobuf(meta *meta.Options, name string) *Protobuf {
	meta.BuildArgs = append(
		meta.BuildArgs,
		"PROTOBUF_GRPC_GATEWAY_TS_VERSION",
	)

	return &Protobuf{
		BaseNode: dag.NewBaseNode(name),

		meta: meta,

		ProtobufTSGatewayVersion: config.ProtobufTSGatewayVersion,

		BaseSpecPath: "/api",
	}
}

// CompileMakefile implements makefile.Compiler.
func (proto *Protobuf) CompileMakefile(output *makefile.Output) error {
	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable("PROTOBUF_GRPC_GATEWAY_TS_VERSION", strings.TrimLeft(proto.ProtobufTSGatewayVersion, "v")))

	return nil
}

// enabled reports whether there are specs to compile.
func (proto *Protobuf) enabled() bool {
	return len(proto.Specs) > 0
}

// compileContainer is the stage that holds the compiled specs, under the node's name.
func (proto *Protobuf) compileContainer() string {
	return "proto-compile-" + proto.Name()
}

// ToolchainPackages implements common.ToolchainPackager hook.
func (proto *Protobuf) ToolchainPackages() []string {
	if !proto.enabled() {
		return nil
	}

	// go installs the protoc plugin, see ToolchainBuild.
	return []string{"go", "protoc", "protobuf-dev"}
}

// ToolchainBuild implements common.ToolchainBuilder hook.
func (proto *Protobuf) ToolchainBuild(stage *dockerfile.Stage) error {
	if !proto.enabled() {
		return nil
	}

	stage.
		Step(step.Arg("PROTOBUF_GRPC_GATEWAY_TS_VERSION")).
		Step(
			step.Script("go install github.com/siderolabs/protoc-gen-grpc-gateway-ts@v${PROTOBUF_GRPC_GATEWAY_TS_VERSION}").
				MountCache(filepath.Join(proto.meta.CachePath, "go-build"), proto.meta.GitHubRepository).
				MountCache(filepath.Join(proto.meta.GoPath, "pkg"), proto.meta.GitHubRepository),
		).
		Step(step.Run("mv", filepath.Join(proto.meta.GoPath, "bin", "protoc-gen-grpc-gateway-ts"), proto.meta.BinPath))

	return nil
}

// CompileDockerfile implements dockerfile.Compiler.
func (proto *Protobuf) CompileDockerfile(output *dockerfile.Output) error {
	if !proto.enabled() {
		return nil
	}

	rootDir := "/" + proto.Name()
	specsContainer := "proto-specs-" + proto.Name()
	compileContainer := proto.compileContainer()

	specs := output.Stage(specsContainer).
		Description("collects proto specs").
		From("scratch")

	for _, spec := range proto.Specs {
		destRoot := proto.DestinationRoot
		if spec.DestinationRoot != "" {
			destRoot = spec.DestinationRoot
		}

		specs.Step(
			step.Add(spec.Source, filepath.Join(rootDir, destRoot, spec.SubDirectory)+"/"),
		)
	}

	compile := output.Stage(compileContainer).
		Description("runs protobuf compiler").
		From("js").
		Step(step.Copy("/", "/").From(specsContainer))

	var cleanupSteps []*step.RunStep

	for _, spec := range proto.Specs {
		destRoot := proto.DestinationRoot
		if spec.DestinationRoot != "" {
			destRoot = spec.DestinationRoot
		}

		dir := filepath.Join(rootDir, destRoot)
		source := filepath.Join(dir, spec.SubDirectory, filepath.Base(spec.Source))

		//nolint:prealloc
		args := []string{
			"-I" + dir,
		}

		args = append(
			args,
			"--grpc-gateway-ts_out=source_relative:"+dir,
			"--grpc-gateway-ts_opt=use_proto_names=true",
		)
		args = append(args, proto.ExperimentalFlags...)
		args = append(args, source)

		compile.Step(
			step.Run(
				"protoc",
				args...,
			),
		)

		if !strings.HasPrefix(spec.Source, "http") {
			cleanupSteps = append(
				cleanupSteps,
				step.Script("rm "+source),
			)
		}
	}

	for _, s := range cleanupSteps {
		compile.Step(s)
	}

	return nil
}

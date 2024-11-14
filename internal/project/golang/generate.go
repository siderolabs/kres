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
	"github.com/siderolabs/kres/internal/output/dockerignore"
	"github.com/siderolabs/kres/internal/output/license"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/output/template"
	"github.com/siderolabs/kres/internal/project/golang/templates"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Generate provides .proto compilation with grpc-go plugin
// and go generate runner.
//
//nolint:govet
type Generate struct {
	dag.BaseNode

	meta *meta.Options

	ProtobufGoVersion  string `yaml:"protobufGoVersion"`
	GrpcGoVersion      string `yaml:"grpcGoVersion"`
	GrpcGatewayVersion string `yaml:"grpcGatewayVersion"`
	VTProtobufVersion  string `yaml:"vtProtobufVersion"`
	GoImportsVersion   string `yaml:"goImportsVersion"`

	BaseSpecPath string `yaml:"baseSpecPath"`

	Specs           []ProtoSpec      `yaml:"specs"`
	GoGenerateSpecs []GoGenerateSpec `yaml:"goGenerateSpecs"`

	ExperimentalFlags []string `yaml:"experimentalFlags"`
	Files             []File   `yaml:"files"`

	VTProtobufEnabled bool `yaml:"vtProtobufEnabled"`

	VersionPackagePath string `yaml:"versionPackagePath"`

	LicenseText string `yaml:"licenseText"`
}

// File represents a file to be fetched/copied into the image.
type File struct {
	Source      string `yaml:"source"`
	Destination string `yaml:"destination"`
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
		"GOIMPORTS_VERSION",
	)

	return &Generate{
		BaseNode: dag.NewBaseNode("protobuf"),

		meta: meta,

		ProtobufGoVersion:  config.ProtobufGoVersion,
		GrpcGoVersion:      config.GrpcGoVersion,
		GrpcGatewayVersion: config.GrpcGatewayVersion,
		VTProtobufVersion:  config.VTProtobufVersion,
		GoImportsVersion:   config.GoImportsVersion,

		BaseSpecPath: "/api",
	}
}

// CompileMakefile implements makefile.Compiler.
func (generate *Generate) CompileMakefile(output *makefile.Output) error {
	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable("PROTOBUF_GO_VERSION", strings.TrimLeft(generate.ProtobufGoVersion, "v"))).
		Variable(makefile.OverridableVariable("GRPC_GO_VERSION", strings.TrimLeft(generate.GrpcGoVersion, "v"))).
		Variable(makefile.OverridableVariable("GRPC_GATEWAY_VERSION", strings.TrimLeft(generate.GrpcGatewayVersion, "v"))).
		Variable(makefile.OverridableVariable("VTPROTOBUF_VERSION", strings.TrimLeft(generate.VTProtobufVersion, "v"))).
		Variable(makefile.OverridableVariable("GOIMPORTS_VERSION", strings.TrimLeft(generate.GoImportsVersion, "v")))

	if len(generate.Specs) == 0 && len(generate.GoGenerateSpecs) == 0 && generate.versionPackagePath() == "" {
		return nil
	}

	output.Target("generate").Description("Generate .proto definitions.").
		Script(`@$(MAKE) local-$@ TARGET_ARGS="--build-arg=BUILDKIT_MULTI_PLATFORM=0 $(TARGET_ARGS)" DEST=./`)

	return nil
}

// ToolchainBuild implements common.ToolchainBuilder hook.
func (generate *Generate) ToolchainBuild(stage *dockerfile.Stage) error {
	if len(generate.Specs) == 0 && len(generate.GoGenerateSpecs) == 0 {
		return nil
	}

	stage.
		Step(step.Arg("GOIMPORTS_VERSION")).
		Step(step.Script("go install golang.org/x/tools/cmd/goimports@v${GOIMPORTS_VERSION}").
			MountCache(filepath.Join(generate.meta.CachePath, "go-build")).
			MountCache(filepath.Join(generate.meta.GoPath, "pkg")),
		).
		Step(step.Run("mv", filepath.Join(generate.meta.GoPath, "bin", "goimports"), generate.meta.BinPath))

	if len(generate.Specs) == 0 {
		// only protobuf stuff after this point
		return nil
	}

	stage.
		Step(step.Arg("PROTOBUF_GO_VERSION")).
		Step(step.Script("go install google.golang.org/protobuf/cmd/protoc-gen-go@v${PROTOBUF_GO_VERSION}").
			MountCache(filepath.Join(generate.meta.CachePath, "go-build")).
			MountCache(filepath.Join(generate.meta.GoPath, "pkg")),
		).
		Step(step.Run("mv", filepath.Join(generate.meta.GoPath, "bin", "protoc-gen-go"), generate.meta.BinPath)).
		Step(step.Arg("GRPC_GO_VERSION")).
		Step(step.Script("go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v${GRPC_GO_VERSION}").
			MountCache(filepath.Join(generate.meta.CachePath, "go-build")).
			MountCache(filepath.Join(generate.meta.GoPath, "pkg")),
		).
		Step(step.Run("mv", filepath.Join(generate.meta.GoPath, "bin", "protoc-gen-go-grpc"), generate.meta.BinPath)).
		Step(step.Arg("GRPC_GATEWAY_VERSION")).
		Step(step.Script("go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v${GRPC_GATEWAY_VERSION}").
			MountCache(filepath.Join(generate.meta.CachePath, "go-build")).
			MountCache(filepath.Join(generate.meta.GoPath, "pkg")),
		).
		Step(step.Run("mv", filepath.Join(generate.meta.GoPath, "bin", "protoc-gen-grpc-gateway"), generate.meta.BinPath))

	if generate.VTProtobufEnabled {
		stage.
			Step(step.Arg("VTPROTOBUF_VERSION")).
			Step(step.Script("go install github.com/planetscale/vtprotobuf/cmd/protoc-gen-go-vtproto@v${VTPROTOBUF_VERSION}").
				MountCache(filepath.Join(generate.meta.CachePath, "go-build")).
				MountCache(filepath.Join(generate.meta.GoPath, "pkg")),
			).
			Step(step.Run("mv", filepath.Join(generate.meta.GoPath, "bin", "protoc-gen-go-vtproto"), generate.meta.BinPath))
	}

	return nil
}

// CompileDockerignore implements dockerignore.Compiler.
func (generate *Generate) CompileDockerignore(output *dockerignore.Output) error {
	if len(generate.GoGenerateSpecs) > 0 {
		output.AllowLocalPath(license.Header)
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

	for _, file := range generate.Files {
		generateStage.Step(step.Add(file.Source, file.Destination))
	}

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
				"-I" + generate.BaseSpecPath,
			}

			if spec.GenGateway {
				flags = append(flags,
					"--grpc-gateway_out=paths=source_relative:"+generate.BaseSpecPath,
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
					"--go_out=paths=source_relative:"+generate.BaseSpecPath,
					"--go-grpc_out=paths=source_relative:"+generate.BaseSpecPath,
				)

				if generate.VTProtobufEnabled {
					flags = append(flags,
						"--go-vtproto_out=paths=source_relative:"+generate.BaseSpecPath,
						"--go-vtproto_opt=features=marshal+unmarshal+size+equal+clone",
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
				strings.Join(generate.meta.CanonicalPaths, ","),
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
				strings.Join(generate.meta.CanonicalPaths, ","),
				spec.Source,
			))
	}

	for index, spec := range generate.GoGenerateSpecs {
		for _, path := range spec.Copy {
			path = filepath.Clean(path)
			generateStage.Step(step.Copy(filepath.Join("/src", path), path).From(fmt.Sprintf("go-generate-%d", index)))
		}
	}

	if generate.versionPackagePath() != "" {
		output.Stage("embed-generate").From("tools").
			Step(step.Arg("SHA")).
			Step(step.Arg("TAG")).
			Step(step.WorkDir("/src")).
			Step(step.Script(fmt.Sprintf(command, generate.versionPackagePath())))

		output.Stage("embed-abbrev-generate").From("embed-generate").
			Step(step.WorkDir("/src")).
			Step(step.Arg("ABBREV_TAG")).
			Step(step.Script(fmt.Sprintf(abbrevCommand, generate.versionPackagePath())))

		src := "/src/" + generate.versionPackagePath()

		generateStage.Step(step.Copy(src, generate.versionPackagePath()).From("embed-abbrev-generate"))

		generate.meta.VersionPackagePath = generate.versionPackagePath()
	}

	return nil
}

// CompileTemplates implements [template.Compiler].
func (generate *Generate) CompileTemplates(output *template.Output) error {
	if generate.versionPackagePath() == "" {
		return nil
	}

	output.Define(generate.versionPackagePath()+"/version.go", templates.VersionGo).
		PreamblePrefix("// ").
		WithLicense().
		WithLicenseText(generate.LicenseText).
		NoOverwrite()

	return nil
}

func (generate *Generate) versionPackagePath() string {
	return strings.TrimSpace(generate.VersionPackagePath)
}

const (
	command = `mkdir -p %[1]s/data && \
    echo -n ${SHA} > %[1]s/data/sha && \
    echo -n ${TAG} > %[1]s/data/tag`

	abbrevCommand = `echo -n 'undefined' > %[1]s/data/sha && \
    echo -n ${ABBREV_TAG} > %[1]s/data/tag`
)

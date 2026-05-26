// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang

import (
	"fmt"
	"path/filepath"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// SBOM filenames produced by the `sbom` stage and shipped as release artifacts.
const (
	sbomSPDXFile      = "sbom.spdx.json"
	sbomCycloneDXFile = "sbom.cyclonedx.json"
)

// SBOM generates a software bill of materials from the project's Go modules and ships it as a release artifact.
type SBOM struct { //nolint:govet
	dag.BaseNode

	meta *meta.Options

	// Enabled turns SBOM generation on.
	Enabled bool `yaml:"enabled"`
	// Version is the syft version to install; defaults to the kres-pinned version.
	Version string `yaml:"version"`
	// SourceName names the SBOM document; defaults to the repository name.
	SourceName string `yaml:"sourceName"`
}

// NewSBOM builds the SBOM node.
func NewSBOM(meta *meta.Options) *SBOM {
	meta.BuildArgs = append(meta.BuildArgs, "SYFT_VERSION")

	return &SBOM{
		BaseNode: dag.NewBaseNode("sbom"),

		meta: meta,

		Version: config.SyftVersion,
	}
}

func (sbom *SBOM) sourceName() string {
	if sbom.SourceName != "" {
		return sbom.SourceName
	}

	return sbom.meta.GitHubRepository
}

// ToolchainBuild implements common.ToolchainBuilder hook: installs syft into the shared tools stage.
func (sbom *SBOM) ToolchainBuild(stage *dockerfile.Stage) error {
	if !sbom.Enabled {
		return nil
	}

	stage.
		Step(step.Arg("SYFT_VERSION")).
		Step(
			step.Script(fmt.Sprintf(
				`go install github.com/anchore/syft/cmd/syft@${SYFT_VERSION} \
	&& mv /go/bin/syft %s/syft`, sbom.meta.BinPath,
			)).
				MountCache(filepath.Join(sbom.meta.CachePath, "go-build"), sbom.meta.GitHubRepository).
				MountCache(filepath.Join(sbom.meta.GoPath, "pkg"), sbom.meta.GitHubRepository),
		)

	return nil
}

// CompileMakefile implements makefile.Compiler.
func (sbom *SBOM) CompileMakefile(output *makefile.Output) error {
	if !sbom.Enabled {
		return nil
	}

	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable("SYFT_VERSION", sbom.Version))

	output.Target("sbom").
		Description("Generate SBOM.").
		Script("@$(MAKE) local-sbom DEST=$(ARTIFACTS)").
		Phony()

	return nil
}

// CompileDockerfile implements dockerfile.Compiler.
func (sbom *SBOM) CompileDockerfile(output *dockerfile.Output) error {
	if !sbom.Enabled {
		return nil
	}

	output.Stage("sbom-generate").
		Description("generates the SBOM").
		From("base").
		Step(step.Arg("TAG")).
		Step(step.WorkDir("/src")).
		Step(
			step.Script(fmt.Sprintf(
				`SYFT_FORMAT_PRETTY=1 SYFT_FORMAT_SPDX_JSON_DETERMINISTIC_UUID=1 syft scan dir:/src --source-name %s --source-version "${TAG}" -o spdx-json=/%s -o cyclonedx-json=/%s`,
				sbom.sourceName(), sbomSPDXFile, sbomCycloneDXFile,
			)),
		)

	output.Stage("sbom").
		From("scratch").
		Step(step.Copy("/"+sbomSPDXFile, "/"+sbomSPDXFile).From("sbom-generate")).
		Step(step.Copy("/"+sbomCycloneDXFile, "/"+sbomCycloneDXFile).From("sbom-generate"))

	return nil
}

// CompileGitHubWorkflow implements ghworkflow.Compiler.
func (sbom *SBOM) CompileGitHubWorkflow(output *ghworkflow.Output) error {
	if !sbom.Enabled {
		return nil
	}

	sbomStep := ghworkflow.Step("sbom").SetMakeStep("sbom")

	if err := sbomStep.SetConditions("only-on-tag"); err != nil {
		return err
	}

	output.AddStep(ghworkflow.DefaultJobName, sbomStep)

	return nil
}

// ReleaseArtifacts implements common.ReleaseArtifactsProvider: the SBOM files are uploaded with the release.
func (sbom *SBOM) ReleaseArtifacts() []string {
	if !sbom.Enabled {
		return nil
	}

	return []string{sbomSPDXFile, sbomCycloneDXFile}
}

// SkipAsMakefileDependency implements makefile.SkipAsMakefileDependency.
func (sbom *SBOM) SkipAsMakefileDependency() {}

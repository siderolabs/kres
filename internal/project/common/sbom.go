// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

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

// golangStdlibCycloneDX is the printf template for the Go standard library SBOM, filled with the toolchain version.
const golangStdlibCycloneDX = `{"bomFormat":"CycloneDX","specVersion":"1.6","components":[{` +
	`"type":"library","name":"stdlib","version":"go%s","purl":"pkg:golang/stdlib@%s",` +
	`"cpe":"cpe:2.3:a:golang:go:%s:-:*:*:*:*:*:*","licenses":[{"license":{"id":"BSD-3-Clause"}}]}]}`

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
	// IncludeStage is a Dockerfile stage holding SBOM documents (SPDX, CycloneDX or syft JSON) to merge into the generated SBOM,
	// e.g. a custom step collecting the /usr/share/spdx files shipped by siderolabs/pkgs. Its contents are placed under the
	// scanned source directory; only files named like an SBOM (e.g. *.spdx.json, *.cdx.json) are picked up.
	//
	// SBOM-named files in the project's own source tree (e.g. test fixtures) are merged as well.
	IncludeStage string `yaml:"includeStage"`
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

	stage := output.Stage("sbom-generate").
		Description("generates the SBOM").
		From("base").
		Step(step.Arg("TAG")).
		Step(step.WorkDir("/src"))

	if sbom.meta.JSEnabled {
		stage.
			Step(step.Copy("/src/package.json", "/src/frontend/package.json").From("js")).
			Step(step.Copy("/src/package-lock.json", "/src/frontend/package-lock.json").From("js"))
	}

	if sbom.IncludeStage != "" {
		stage.Step(step.Copy("/", "/src/").From(sbom.IncludeStage))
	}

	// go.mod doesn't list the Go standard library compiled into the binaries, so describe it from the toolchain
	// in this stage (the same one the binaries are built with), matching the package syft emits for Go binaries
	stage.Step(step.Script(`GOVERSION="$(go env GOVERSION)" && GOVERSION="${GOVERSION#go}" \
	&& printf '` + golangStdlibCycloneDX + `' "${GOVERSION}" "${GOVERSION}" "${GOVERSION}" > /src/golang-stdlib.cdx.json`))

	// the SBOM cataloger is off by default for directory scans, enable it to merge the stdlib and included documents;
	// it also merges any SBOM-named files in the source tree
	stage.Step(
		step.Script(fmt.Sprintf(
			`SYFT_FORMAT_PRETTY=1 SYFT_FORMAT_SPDX_JSON_DETERMINISTIC_UUID=1 SYFT_GOLANG_SEARCH_LOCAL_MOD_CACHE_LICENSES=1 SYFT_GOLANG_LOCAL_MOD_CACHE_DIR=%s `+
				`syft scan dir:/src --select-catalogers +sbom-cataloger --source-name %s --source-version "${TAG}" -o spdx-json=/%s -o cyclonedx-json=/%s`,
			filepath.Join(sbom.meta.GoPath, "pkg", "mod"), sbom.sourceName(), sbomSPDXFile, sbomCycloneDXFile,
		)).
			MountCache(filepath.Join(sbom.meta.CachePath, "go-build"), sbom.meta.GitHubRepository).
			MountCache(filepath.Join(sbom.meta.GoPath, "pkg"), sbom.meta.GitHubRepository),
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

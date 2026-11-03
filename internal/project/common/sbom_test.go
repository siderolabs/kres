// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/common"
	"github.com/siderolabs/kres/internal/project/meta"
)

func TestSBOMInterfaces(t *testing.T) {
	assert.Implements(t, (*common.ToolchainBuilder)(nil), new(common.SBOM))
	assert.Implements(t, (*dockerfile.Compiler)(nil), new(common.SBOM))
	assert.Implements(t, (*makefile.Compiler)(nil), new(common.SBOM))
	assert.Implements(t, (*ghworkflow.Compiler)(nil), new(common.SBOM))
	assert.Implements(t, (*common.ReleaseArtifactsProvider)(nil), new(common.SBOM))
	assert.Implements(t, (*makefile.SkipAsMakefileDependency)(nil), new(common.SBOM))
}

func sbomDockerfile(t *testing.T, opts *meta.Options, includeStage string) string {
	t.Helper()

	sbom := common.NewSBOM(opts)
	sbom.Enabled = true
	sbom.IncludeStage = includeStage

	var output dockerfile.Output

	require.NoError(t, sbom.CompileDockerfile(&output))

	var buf bytes.Buffer

	require.NoError(t, output.GenerateFile("Dockerfile", &buf))

	return buf.String()
}

func TestSBOMDockerfileGoOnly(t *testing.T) {
	rendered := sbomDockerfile(t, &meta.Options{
		CachePath:        "/root/.cache",
		GitHubRepository: "example",
		GoPath:           "/go",
	}, "")

	require.Contains(t, rendered, "syft scan dir:/src --select-catalogers +sbom-cataloger --source-name example")
	// No JS component: the frontend manifests must not be copied in.
	require.NotContains(t, rendered, "--from=js")
	// No included SBOMs.
	require.NotContains(t, rendered, "/ /src/")
	// The Go standard library is described from the toolchain.
	require.Contains(t, rendered, `GOVERSION="$(go env GOVERSION)"`)
	require.Contains(t, rendered, "> /src/golang-stdlib.cdx.json\n")
}

func TestSBOMDockerfileWithJS(t *testing.T) {
	rendered := sbomDockerfile(t, &meta.Options{
		CachePath:        "/root/.cache",
		GitHubRepository: "example",
		GoPath:           "/go",
		JSEnabled:        true,
	}, "")

	require.Contains(t, rendered, "COPY --from=js /src/package.json /src/frontend/package.json\n")
	require.Contains(t, rendered, "COPY --from=js /src/package-lock.json /src/frontend/package-lock.json\n")
	// The combined scan and its output filenames are unchanged.
	require.Contains(t, rendered, "syft scan dir:/src --select-catalogers +sbom-cataloger --source-name example")
}

func TestSBOMDockerfileWithInclude(t *testing.T) {
	rendered := sbomDockerfile(t, &meta.Options{
		CachePath:        "/root/.cache",
		GitHubRepository: "example",
		GoPath:           "/go",
	}, "sboms")

	require.Contains(t, rendered, "COPY --from=sboms / /src/\n")
	require.Contains(t, rendered, "syft scan dir:/src --select-catalogers +sbom-cataloger --source-name example")
}

func TestSBOMDockerfileStdlibIsValidJSON(t *testing.T) {
	rendered := sbomDockerfile(t, &meta.Options{GitHubRepository: "example"}, "")

	match := regexp.MustCompile(`printf '([^']+)'`).FindStringSubmatch(rendered)
	require.Len(t, match, 2)

	var doc struct {
		Components []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			PURL    string `json:"purl"`
			CPE     string `json:"cpe"`
		} `json:"components"`
	}

	require.NoError(t, json.Unmarshal([]byte(fmt.Sprintf(match[1], "1.27.3", "1.27.3", "1.27.3")), &doc))
	require.Len(t, doc.Components, 1)
	assert.Equal(t, "stdlib", doc.Components[0].Name)
	assert.Equal(t, "go1.27.3", doc.Components[0].Version)
	assert.Equal(t, "pkg:golang/stdlib@1.27.3", doc.Components[0].PURL)
	assert.Equal(t, "cpe:2.3:a:golang:go:1.27.3:-:*:*:*:*:*:*", doc.Components[0].CPE)
}

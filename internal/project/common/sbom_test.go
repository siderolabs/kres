// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common_test

import (
	"bytes"
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

func sbomDockerfile(t *testing.T, opts *meta.Options) string {
	t.Helper()

	sbom := common.NewSBOM(opts)
	sbom.Enabled = true

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
	})

	require.Contains(t, rendered, "syft scan dir:/src --source-name example")
	// No JS component: the frontend manifests must not be copied in.
	require.NotContains(t, rendered, "--from=js")
}

func TestSBOMDockerfileWithJS(t *testing.T) {
	rendered := sbomDockerfile(t, &meta.Options{
		CachePath:        "/root/.cache",
		GitHubRepository: "example",
		GoPath:           "/go",
		JSEnabled:        true,
	})

	require.Contains(t, rendered, "COPY --from=js /src/package.json /src/frontend/package.json\n")
	require.Contains(t, rendered, "COPY --from=js /src/package-lock.json /src/frontend/package-lock.json\n")
	// The combined scan and its output filenames are unchanged.
	require.Contains(t, rendered, "syft scan dir:/src --source-name example")
}

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/common"
	"github.com/siderolabs/kres/internal/project/golang"
	"github.com/siderolabs/kres/internal/project/meta"
)

func TestToolchainInterfaces(t *testing.T) {
	assert.Implements(t, (*dockerfile.Compiler)(nil), new(golang.Toolchain))
	assert.Implements(t, (*makefile.Compiler)(nil), new(golang.Toolchain))
	assert.Implements(t, (*ghworkflow.Compiler)(nil), new(golang.Toolchain))
	assert.Implements(t, (*makefile.SkipAsMakefileDependency)(nil), new(golang.Toolchain))
}

func TestToolchainSourceAssets(t *testing.T) {
	assets := common.NewSourceAssets(&meta.Options{})
	assets.Images = []common.SourceAssetsImage{{
		Ref: "ghcr.io/siderolabs/ipxe:v1.11.0",
		Copies: []common.SourceAssetsCopy{
			{Platform: "linux/amd64", Source: "/usr/libexec/", Destination: "internal/ipxe/data/amd64"},
			{Platform: "linux/arm64", Source: "/usr/libexec/", Destination: "internal/ipxe/data/arm64"},
		},
	}}
	require.NoError(t, assets.AfterLoad())

	toolchain := golang.NewToolchain(&meta.Options{
		GoContainerVersion: "1.26",
		Directories:        []string{"internal"},
		GoRootDirectories:  []string{"."},
		GoDirectories:      []string{"internal"},
	})
	toolchain.AddInput(assets)

	output := dockerfile.NewOutput()

	require.NoError(t, assets.CompileDockerfile(output))
	require.NoError(t, toolchain.CompileDockerfile(output))

	var buf bytes.Buffer

	require.NoError(t, output.GenerateFile("Dockerfile", &buf))

	generated := buf.String()

	assert.Contains(t, generated, "ghcr.io/siderolabs/ipxe:v1.11.0 AS source-assets-src-0")
	assert.Contains(t, generated, "ghcr.io/siderolabs/ipxe:v1.11.0 AS source-assets-src-1")
	assert.Contains(t, generated, "COPY --from=source-assets-src-0 /usr/libexec/ /internal/ipxe/data/amd64")

	// the collector contents land in the source tree of base, before any Go package loading
	copyLine := "COPY --from=source-assets / ./"
	assert.Contains(t, generated, copyLine)
	assert.Less(t, strings.Index(generated, copyLine), strings.Index(generated, "go list -mod=readonly"))
}

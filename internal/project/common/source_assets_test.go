// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common_test

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerignore"
	"github.com/siderolabs/kres/internal/output/gitignore"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/output/renovate"
	"github.com/siderolabs/kres/internal/project/common"
	"github.com/siderolabs/kres/internal/project/meta"
)

func sourceAssets(ref string, copies ...common.SourceAssetsCopy) *common.SourceAssets {
	assets := common.NewSourceAssets(&meta.Options{})
	assets.Images = []common.SourceAssetsImage{{Ref: ref, Copies: copies}}

	return assets
}

func TestSourceAssetsValidation(t *testing.T) {
	for _, tt := range []struct {
		name     string
		assets   *common.SourceAssets
		wantErr  string
		wantDest string
	}{
		{
			name:     "valid",
			assets:   sourceAssets("ghcr.io/siderolabs/ipxe:v1.11.0", common.SourceAssetsCopy{Source: "/usr/libexec/", Destination: "internal/data"}),
			wantDest: "internal/data",
		},
		{
			name:     "normalized",
			assets:   sourceAssets("ghcr.io/siderolabs/ipxe:v1.11.0", common.SourceAssetsCopy{Source: "/x", Destination: "./internal//data/"}),
			wantDest: "internal/data",
		},
		{
			name:    "empty ref",
			assets:  sourceAssets("", common.SourceAssetsCopy{Source: "/x", Destination: "internal/data"}),
			wantErr: "ref is empty",
		},
		{
			name:    "no copies",
			assets:  sourceAssets("ghcr.io/siderolabs/ipxe:v1.11.0"),
			wantErr: "declares no copies",
		},
		{
			name:    "relative source",
			assets:  sourceAssets("ghcr.io/siderolabs/ipxe:v1.11.0", common.SourceAssetsCopy{Source: "usr/libexec/", Destination: "internal/data"}),
			wantErr: "must be an absolute path",
		},
		{
			name:    "absolute destination",
			assets:  sourceAssets("ghcr.io/siderolabs/ipxe:v1.11.0", common.SourceAssetsCopy{Source: "/x", Destination: "/etc"}),
			wantErr: "must be a relative path",
		},
		{
			name:    "escaping destination",
			assets:  sourceAssets("ghcr.io/siderolabs/ipxe:v1.11.0", common.SourceAssetsCopy{Source: "/x", Destination: "../outside"}),
			wantErr: "must be a relative path",
		},
		{
			name: "duplicate destination",
			assets: sourceAssets(
				"ghcr.io/siderolabs/ipxe:v1.11.0",
				common.SourceAssetsCopy{Source: "/x", Destination: "internal/data"},
				common.SourceAssetsCopy{Source: "/y", Destination: "internal/data"},
			),
			wantErr: "overlap",
		},
		{
			name: "nested destination",
			assets: sourceAssets(
				"ghcr.io/siderolabs/ipxe:v1.11.0",
				common.SourceAssetsCopy{Source: "/x", Destination: "internal/data"},
				common.SourceAssetsCopy{Source: "/y", Destination: "internal/data/nested"},
			),
			wantErr: "overlap",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.assets.AfterLoad()

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantDest, tt.assets.Images[0].Copies[0].Destination)
		})
	}
}

func TestSourceAssetsOutputs(t *testing.T) {
	assets := sourceAssets(
		"ghcr.io/siderolabs/ipxe:v1.11.0",
		common.SourceAssetsCopy{Platform: "linux/amd64", Source: "/usr/libexec/", Destination: "internal/data/amd64"},
		common.SourceAssetsCopy{Platform: "linux/arm64", Source: "/usr/libexec/", Destination: "internal/data/arm64"},
	)
	require.NoError(t, assets.AfterLoad())

	t.Run("makefile", func(t *testing.T) {
		o := makefile.NewOutput()
		require.NoError(t, assets.CompileMakefile(o))

		var buf bytes.Buffer
		require.NoError(t, o.GenerateFile("Makefile", &buf))

		assert.Contains(t, buf.String(), "fetch-source-assets:")
		assert.Contains(t, buf.String(), "rm -rf -- 'internal/data/amd64' 'internal/data/arm64'")
		assert.Contains(t, buf.String(), "$(MAKE) local-source-assets DEST=./ PLATFORM=linux/amd64")
	})

	t.Run("gitignore", func(t *testing.T) {
		o := gitignore.NewOutput()
		require.NoError(t, assets.CompileGitignore(o))

		var buf bytes.Buffer
		require.NoError(t, o.GenerateFile(".gitignore", &buf))

		assert.Contains(t, buf.String(), "internal/data/amd64")
		assert.Contains(t, buf.String(), "internal/data/arm64")
	})

	t.Run("dockerignore", func(t *testing.T) {
		o := dockerignore.NewOutput()
		o.AllowLocalPath("internal")
		require.NoError(t, assets.CompileDockerignore(o))

		var buf bytes.Buffer
		require.NoError(t, o.GenerateFile(".dockerignore", &buf))

		content := buf.String()
		assert.Contains(t, content, "!internal\n")
		// the deny entries come after the allows, so they win
		assert.Greater(t, bytes.Index(buf.Bytes(), []byte("internal/data/amd64")), bytes.Index(buf.Bytes(), []byte("!internal\n")))
	})
}

func TestSourceAssetsRenovateRegex(t *testing.T) {
	o := renovate.NewOutput()
	o.Enable()

	assets := sourceAssets("ghcr.io/siderolabs/ipxe:v1.11.0", common.SourceAssetsCopy{Source: "/x", Destination: "internal/data"})
	require.NoError(t, assets.CompileRenovate(o))

	var buf bytes.Buffer
	require.NoError(t, o.GenerateFile(".github/renovate.json", &buf))

	var config struct {
		CustomManagers []struct {
			MatchStrings []string `json:"matchStrings"`
		} `json:"customManagers"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &config))
	require.Len(t, config.CustomManagers, 1)
	require.Len(t, config.CustomManagers[0].MatchStrings, 1)

	re := regexp.MustCompile(config.CustomManagers[0].MatchStrings[0])

	for _, tt := range []struct {
		line    string
		wantDep string
		wantVer string
	}{
		{"      ref: ghcr.io/siderolabs/ipxe:v1.11.0", "ghcr.io/siderolabs/ipxe", "v1.11.0"},
		{`      ref: "ghcr.io/siderolabs/ipxe:v1.11.0"`, "ghcr.io/siderolabs/ipxe", "v1.11.0"},
		{"      ref: registry.internal:5000/siderolabs/ipxe:v1.2.3", "registry.internal:5000/siderolabs/ipxe", "v1.2.3"},
	} {
		m := re.FindStringSubmatch(tt.line)
		require.NotNil(t, m, tt.line)
		assert.Equal(t, tt.wantDep, m[re.SubexpIndex("depName")], tt.line)
		assert.Equal(t, tt.wantVer, m[re.SubexpIndex("currentValue")], tt.line)
	}
}

func TestSourceAssetsStageDedup(t *testing.T) {
	assets := common.NewSourceAssets(&meta.Options{})
	assets.Images = []common.SourceAssetsImage{
		{Ref: "ghcr.io/siderolabs/ipxe:v1.11.0", Copies: []common.SourceAssetsCopy{{Platform: "linux/amd64", Source: "/a", Destination: "internal/a"}}},
		{Ref: "ghcr.io/siderolabs/ipxe:v1.11.0", Copies: []common.SourceAssetsCopy{{Platform: "linux/amd64", Source: "/b", Destination: "internal/b"}}},
	}
	require.NoError(t, assets.AfterLoad())

	o := dockerfile.NewOutput()
	o.Enable()
	require.NoError(t, assets.CompileDockerfile(o))

	var buf bytes.Buffer
	require.NoError(t, o.GenerateFile("Dockerfile", &buf))

	// the identical image and platform across separate entries collapse into one source stage
	assert.Equal(t, 1, strings.Count(buf.String(), "AS source-assets-src-"))
	assert.Contains(t, buf.String(), "COPY --from=source-assets-src-0 /a /internal/a")
	assert.Contains(t, buf.String(), "COPY --from=source-assets-src-0 /b /internal/b")
}

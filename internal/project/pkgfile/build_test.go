// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package pkgfile_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
	"github.com/siderolabs/kres/internal/project/pkgfile"
)

func TestBuildInterfaces(t *testing.T) {
	assert.Implements(t, (*makefile.Compiler)(nil), new(pkgfile.Build))
}

// TestNewBuildDefaultsBldrImageVersion verifies the node defaults the bldr
// version to the package constant when nothing overrides it.
func TestNewBuildDefaultsBldrImageVersion(t *testing.T) {
	build := pkgfile.NewBuild(&meta.Options{})

	assert.Equal(t, config.BldrImageVersion, build.BldrImageVersion)
}

func compileMakefile(t *testing.T, build *pkgfile.Build) string {
	t.Helper()

	o := makefile.NewOutput()

	require.NoError(t, o.Compile(build))

	var buf bytes.Buffer

	require.NoError(t, o.GenerateFile("Makefile", &buf))

	return buf.String()
}

// TestCompileMakefileDefaultBldrRelease verifies the generated Makefile pins
// BLDR_RELEASE to the default constant when the version is not overridden.
func TestCompileMakefileDefaultBldrRelease(t *testing.T) {
	build := pkgfile.NewBuild(&meta.Options{})

	out := compileMakefile(t, build)

	assert.Contains(t, out, "BLDR_RELEASE := "+config.BldrImageVersion)
	assert.Contains(t, out, "BLDR_IMAGE := ghcr.io/siderolabs/bldr:$(BLDR_RELEASE)")
}

// TestCompileMakefilePinnedBldrRelease verifies a pinned bldr version (as set
// from .kres.yaml) flows into the generated BLDR_RELEASE variable.
func TestCompileMakefilePinnedBldrRelease(t *testing.T) {
	build := pkgfile.NewBuild(&meta.Options{})
	build.BldrImageVersion = "v0.5.0"

	out := compileMakefile(t, build)

	assert.Contains(t, out, "BLDR_RELEASE := v0.5.0")
	assert.NotContains(t, out, "BLDR_RELEASE := "+config.BldrImageVersion)
}

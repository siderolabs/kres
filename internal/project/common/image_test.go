// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common_test

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/common"
	"github.com/siderolabs/kres/internal/project/meta"
)

func TestImageInterfaces(t *testing.T) {
	assert.Implements(t, (*makefile.Compiler)(nil), new(common.Image))
	assert.Implements(t, (*dockerfile.Compiler)(nil), new(common.Image))
	assert.Implements(t, (*ghworkflow.Compiler)(nil), new(common.Image))
}

func renderMakefile(t *testing.T, images ...*common.Image) string {
	t.Helper()

	output := makefile.NewOutput()

	for _, image := range images {
		require.NoError(t, image.CompileMakefile(output))
	}

	var buf bytes.Buffer

	require.NoError(t, output.GenerateFile("Makefile", &buf))

	return buf.String()
}

func TestImageSignTarget(t *testing.T) {
	rendered := renderMakefile(t, common.NewImage(&meta.Options{}, "omni"))

	assert.Contains(t, rendered, "IMAGE_SIGNER_IMAGE ?= ghcr.io/siderolabs/image-signer:")

	assert.Contains(t, rendered, `.PHONY: sign-image-omni
sign-image-omni:  ## Signs the image for omni. Requires interactive Google authentication.
	@test -n "$$GITHUB_TOKEN" || { \
	  echo 'GITHUB_TOKEN with write:packages scope must be set: gh auth refresh -s write:packages, then GITHUB_TOKEN=$$(gh auth token) make' $@; \
	  exit 1; \
	}
	@TMP=$$(mktemp -d) && trap 'rm -rf "$$TMP"' EXIT && \
	  printf '{"auths":{"$(REGISTRY)":{"username":"x","password":"%s"}}}' "$$GITHUB_TOKEN" > "$$TMP/config.json" && \
	  docker run --rm -p 127.0.0.1:8585:8585 \
	    -v "$$TMP:/dc:ro" -e DOCKER_CONFIG=/dc \
	    $(IMAGE_SIGNER_IMAGE) sign --timeout=15m \
	    $(REGISTRY)/$(USERNAME)/omni:$(IMAGE_TAG)
`)
}

// TestImageSignTargetName covers an image whose published name differs from the node name: the
// target pairs with the build target, while the signed reference uses the published name.
func TestImageSignTargetName(t *testing.T) {
	image := common.NewImage(&meta.Options{}, "exporter")
	image.ImageName = "omni-audit-log-exporter"

	rendered := renderMakefile(t, image)

	assert.Contains(t, rendered, ".PHONY: sign-image-exporter\n")
	assert.Contains(t, rendered, "$(REGISTRY)/$(USERNAME)/omni-audit-log-exporter:$(IMAGE_TAG)\n")
}

func TestImageSignTargetMultipleImages(t *testing.T) {
	rendered := renderMakefile(
		t,
		common.NewImage(&meta.Options{}, "omni"),
		common.NewImage(&meta.Options{}, "omni-integration-test"),
	)

	assert.Contains(t, rendered, ".PHONY: sign-image-omni\n")
	assert.Contains(t, rendered, ".PHONY: sign-image-omni-integration-test\n")

	// the signer image variable is shared, it must not be emitted once per image
	assert.Equal(t, 1, strings.Count(rendered, "IMAGE_SIGNER_IMAGE ?="))
}

// TestImageSignTargetNotAutomated guards the rule that images are only ever signed by a human.
// Within what an image compiles: there is no aggregate target, no other target names a sign target
// as a dependency, and the CI workflow never runs one.
func TestImageSignTargetNotAutomated(t *testing.T) {
	images := []*common.Image{
		common.NewImage(&meta.Options{}, "omni"),
		common.NewImage(&meta.Options{}, "omni-integration-test"),
	}

	rendered := renderMakefile(t, images...)

	assert.NotContains(t, rendered, "sign-images")

	for _, name := range []string{"sign-image-omni", "sign-image-omni-integration-test"} {
		assert.Regexp(t, regexp.MustCompile(`(?m)^`+name+`:`), rendered)

		// the name occurs exactly twice, on its own .PHONY line and its own target line. a third
		// occurrence would be a prerequisite reference, wherever in the line it sits. the trailing
		// class keeps the shorter name from matching inside the longer one
		assert.Len(t, regexp.MustCompile(name+`([^-\w]|$)`).FindAllString(rendered, -1), 2,
			"%s is referenced outside of its own definition", name)
	}

	workflow := ghworkflow.NewOutput("main", true, false, "")
	workflow.SetRunnerGroup(ghworkflow.GenericRunner)

	for _, image := range images {
		require.NoError(t, image.CompileGitHubWorkflow(workflow))
	}

	var buf bytes.Buffer

	require.NoError(t, workflow.GenerateFile(ghworkflow.CiWorkflow, &buf))

	assert.NotContains(t, buf.String(), "sign")
}

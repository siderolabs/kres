// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerignore"
	"github.com/siderolabs/kres/internal/project/golang"
	"github.com/siderolabs/kres/internal/project/meta"
)

func TestGenerateExtraInputs(t *testing.T) {
	generate := golang.NewGenerate(&meta.Options{
		CachePath:        "/root/.cache",
		CanonicalPaths:   []string{"github.com/siderolabs/example"},
		GitHubRepository: "example",
		GoPath:           "/go",
	})

	generate.GoGenerateSpecs = []golang.GoGenerateSpec{
		{
			Source: "./internal",
			ExtraInputs: []string{
				"./deploy/helm/omni/config-overrides.yaml",
				"deploy/helm/omni/values.yaml",
			},
		},
	}

	var dockerfileOutput dockerfile.Output

	require.NoError(t, generate.CompileDockerfile(&dockerfileOutput))

	var dockerfileBuffer bytes.Buffer

	require.NoError(t, dockerfileOutput.GenerateFile("Dockerfile", &dockerfileBuffer))

	dockerfile := dockerfileBuffer.String()

	require.Contains(t, dockerfile, "COPY deploy/helm/omni/config-overrides.yaml deploy/helm/omni/config-overrides.yaml\n")
	require.Contains(t, dockerfile, "COPY deploy/helm/omni/values.yaml deploy/helm/omni/values.yaml\n")
	require.Contains(t, dockerfile, "RUN --mount=type=cache,target=/root/.cache/go-build,id=example/root/.cache/go-build "+
		"--mount=type=cache,target=/go/pkg,id=example/go/pkg go generate ./internal/...\n")

	dockerignoreOutput := dockerignore.NewOutput()

	require.NoError(t, generate.CompileDockerignore(dockerignoreOutput))

	var dockerignoreBuffer bytes.Buffer

	require.NoError(t, dockerignoreOutput.GenerateFile(".dockerignore", &dockerignoreBuffer))

	dockerignore := dockerignoreBuffer.String()

	require.Contains(t, dockerignore, "!deploy/helm/omni/config-overrides.yaml\n")
	require.Contains(t, dockerignore, "!deploy/helm/omni/values.yaml\n")
}

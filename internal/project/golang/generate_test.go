// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"

	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerignore"
	"github.com/siderolabs/kres/internal/output/lefthook"
	"github.com/siderolabs/kres/internal/project/golang"
	"github.com/siderolabs/kres/internal/project/meta"
)

func TestGenerateInterfaces(t *testing.T) {
	assert.Implements(t, (*dockerfile.Compiler)(nil), new(golang.Generate))
	assert.Implements(t, (*dockerignore.Compiler)(nil), new(golang.Generate))
	assert.Implements(t, (*lefthook.Compiler)(nil), new(golang.Generate))
}

func TestGenerateLefthook(t *testing.T) {
	generate := golang.NewGenerate(&meta.Options{
		GitHubOrganization: "testorg",
	})

	// `make generate` joins the shared fix stage as a named job.
	output := lefthook.NewOutput()
	output.Enable()

	require.NoError(t, generate.CompileLefthook(output))

	var buf bytes.Buffer

	require.NoError(t, output.GenerateFile("lefthook.yml", &buf))

	rendered := buf.String()
	assert.Contains(t, rendered, "name: generate")
	assert.Contains(t, rendered, "run: make generate")
	assert.Contains(t, rendered, "stage_fixed: true")

	var decoded map[string]struct {
		Jobs []struct {
			Group *struct {
				Jobs []struct {
					Env  map[string]string `yaml:"env"`
					Name string            `yaml:"name"`
				} `yaml:"jobs"`
			} `yaml:"group"`
		} `yaml:"jobs"`
	}

	require.NoError(t, yaml.Unmarshal(buf.Bytes(), &decoded))

	hook := decoded["pre-commit"]
	require.NotEmpty(t, hook.Jobs)
	require.NotNil(t, hook.Jobs[0].Group)

	var generateJob *struct {
		Env  map[string]string `yaml:"env"`
		Name string            `yaml:"name"`
	}

	for i := range hook.Jobs[0].Group.Jobs {
		if hook.Jobs[0].Group.Jobs[i].Name == "generate" {
			generateJob = &hook.Jobs[0].Group.Jobs[i]

			break
		}
	}

	require.NotNil(t, generateJob, "generate job not found")
	assert.Equal(t, "testorg", generateJob.Env["USERNAME"])
}

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

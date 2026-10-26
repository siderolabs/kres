// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package js_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"

	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/lefthook"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/js"
	"github.com/siderolabs/kres/internal/project/meta"
)

func TestGenerateInterfaces(t *testing.T) {
	assert.Implements(t, (*dockerfile.Compiler)(nil), new(js.Generate))
	assert.Implements(t, (*makefile.Compiler)(nil), new(js.Generate))
	assert.Implements(t, (*lefthook.Compiler)(nil), new(js.Generate))
}

func newGenerate(options *meta.Options, specs bool, script string) *js.Generate {
	proto := js.NewProtobuf(options, "frontend")
	if specs {
		proto.Specs = []js.ProtoSpec{{Source: "api.proto"}}
	}

	generate := js.NewGenerate(options, "frontend", proto)
	generate.Script = script

	if script != "" {
		generate.ScriptInputs = []string{"api"}
		generate.ScriptOutputs = []string{"src/api/schema.d.ts"}
	}

	return generate
}

func renderMakefile(t *testing.T, generate *js.Generate, registerGenerate bool) string {
	t.Helper()

	output := makefile.NewOutput()

	if registerGenerate {
		output.Target("generate")
	}

	require.NoError(t, generate.CompileMakefile(output))

	var buf bytes.Buffer

	require.NoError(t, output.GenerateFile("Makefile", &buf))

	return buf.String()
}

func renderLefthook(t *testing.T, generate *js.Generate) []byte {
	t.Helper()

	output := lefthook.NewOutput()
	output.Enable()

	require.NoError(t, generate.CompileLefthook(output))

	var buf bytes.Buffer

	require.NoError(t, output.GenerateFile("lefthook.yml", &buf))

	return buf.Bytes()
}

func TestGenerateMakefileGenerateDepends(t *testing.T) {
	for _, tt := range []struct {
		name             string
		script           string
		registerGenerate bool
		specs            bool
		wantTarget       bool
		wantDepends      bool
	}{
		{
			name:             "protobuf specs: generate target depends on generate-frontend",
			registerGenerate: true,
			specs:            true,
			wantTarget:       true,
			wantDepends:      true,
		},
		{
			name:             "npm script: generate target depends on generate-frontend",
			registerGenerate: true,
			script:           "generate",
			wantTarget:       true,
			wantDepends:      true,
		},
		{
			name:       "no generate target leaves generate-frontend orphan",
			specs:      true,
			wantTarget: true,
		},
		{
			name:             "nothing to generate: no generate-frontend target",
			registerGenerate: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rendered := renderMakefile(t, newGenerate(&meta.Options{}, tt.specs, tt.script), tt.registerGenerate)

			if tt.wantTarget {
				assert.Contains(t, rendered, "generate-frontend:")
			} else {
				assert.NotContains(t, rendered, "generate-frontend")
			}

			if tt.wantDepends {
				assert.Contains(t, rendered, "generate: generate-frontend")
			} else {
				assert.NotContains(t, rendered, "generate: generate-frontend")
			}
		})
	}
}

func TestGenerateMakefileGenerateDependsCreatedLater(t *testing.T) {
	// The frontend compiles before the Go generate target exists, as in a real project.
	output := makefile.NewOutput()

	require.NoError(t, newGenerate(&meta.Options{}, false, "generate").CompileMakefile(output))

	output.Target("generate").Description("Generate .proto definitions.")

	var buf bytes.Buffer

	require.NoError(t, output.GenerateFile("Makefile", &buf))
	assert.Contains(t, buf.String(), "generate: generate-frontend")

	// rendering again doesn't add the dependency twice
	buf.Reset()

	require.NoError(t, output.GenerateFile("Makefile", &buf))
	assert.Contains(t, buf.String(), "generate: generate-frontend  ##")
}

func TestGenerateLefthook(t *testing.T) {
	// `make generate-frontend` joins the shared fix stage as a named job.
	rendered := renderLefthook(t, newGenerate(&meta.Options{GitHubOrganization: "testorg"}, true, ""))

	assert.Contains(t, string(rendered), "name: generate frontend")
	assert.Contains(t, string(rendered), "run: make generate-frontend")
	assert.Contains(t, string(rendered), "stage_fixed: true")

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

	require.NoError(t, yaml.Unmarshal(rendered, &decoded))

	hook := decoded["pre-commit"]
	require.NotEmpty(t, hook.Jobs)
	require.NotNil(t, hook.Jobs[0].Group)

	var generateJob *struct {
		Env  map[string]string `yaml:"env"`
		Name string            `yaml:"name"`
	}

	for i := range hook.Jobs[0].Group.Jobs {
		if hook.Jobs[0].Group.Jobs[i].Name == "generate frontend" {
			generateJob = &hook.Jobs[0].Group.Jobs[i]

			break
		}
	}

	require.NotNil(t, generateJob, "generate frontend job not found")
	assert.Equal(t, "testorg", generateJob.Env["USERNAME"])
}

func TestGenerateLefthookNothingToGenerate(t *testing.T) {
	// A job running a target that doesn't exist would fail every commit.
	assert.NotContains(t, string(renderLefthook(t, newGenerate(&meta.Options{}, false, ""))), "generate-frontend")
}

func TestGenerateDockerfile(t *testing.T) {
	for _, tt := range []struct {
		name    string
		script  string
		want    []string
		notWant []string
		specs   bool
	}{
		{
			name:   "npm script",
			script: "generate",
			want: []string{
				"FROM js AS npm-generate-frontend",
				"COPY api /api",
				"RUN npm run generate",
				"FROM scratch AS generate-frontend",
				"COPY --from=npm-generate-frontend /src/src/api/schema.d.ts frontend/src/api/schema.d.ts",
			},
			notWant: []string{"proto-compile-frontend"},
		},
		{
			name:  "protobuf specs",
			specs: true,
			want: []string{
				"FROM scratch AS generate-frontend",
				"COPY --from=proto-compile-frontend frontend/ frontend/", //nolint:dupword
			},
			notWant: []string{"npm-generate-frontend"},
		},
		{
			name:   "both",
			specs:  true,
			script: "generate",
			want: []string{
				"COPY --from=proto-compile-frontend frontend/ frontend/", //nolint:dupword
				"COPY --from=npm-generate-frontend /src/src/api/schema.d.ts frontend/src/api/schema.d.ts",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			generate := newGenerate(&meta.Options{}, tt.specs, tt.script)

			output := dockerfile.NewOutput()

			require.NoError(t, generate.CompileDockerfile(output))

			var buf bytes.Buffer

			require.NoError(t, output.GenerateFile("Dockerfile", &buf))

			rendered := buf.String()

			for _, want := range tt.want {
				assert.Contains(t, rendered, want)
			}

			for _, notWant := range tt.notWant {
				assert.NotContains(t, rendered, notWant)
			}
		})
	}
}

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

func TestProtobufInterfaces(t *testing.T) {
	assert.Implements(t, (*dockerfile.Compiler)(nil), new(js.Protobuf))
	assert.Implements(t, (*makefile.Compiler)(nil), new(js.Protobuf))
	assert.Implements(t, (*lefthook.Compiler)(nil), new(js.Protobuf))
}

func TestProtobufMakefileGenerateDepends(t *testing.T) {
	for _, tt := range []struct {
		name               string
		registerCheckDirty bool
		wantDepends        bool
	}{
		{
			name:               "generate target depends on generate-frontend",
			registerCheckDirty: true,
			wantDepends:        true,
		},
		{
			name:               "no generate target leaves generate-frontend orphan",
			registerCheckDirty: false,
			wantDepends:        false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			proto := js.NewProtobuf(&meta.Options{}, "frontend")
			proto.Specs = []js.ProtoSpec{{Source: "api.proto"}}

			output := makefile.NewOutput()

			if tt.registerCheckDirty {
				output.Target("generate")
			}

			require.NoError(t, proto.CompileMakefile(output))

			var buf bytes.Buffer

			require.NoError(t, output.GenerateFile("Makefile", &buf))

			rendered := buf.String()
			assert.Contains(t, rendered, "generate-frontend:")

			if tt.wantDepends {
				assert.Contains(t, rendered, "generate: generate-frontend")
			} else {
				assert.NotContains(t, rendered, "generate:")
			}
		})
	}
}

func TestProtobufLefthook(t *testing.T) {
	proto := js.NewProtobuf(&meta.Options{
		GitHubOrganization: "testorg",
	}, "frontend")

	// `make generate-frontend` joins the shared fix stage as a named job.
	output := lefthook.NewOutput()
	output.Enable()

	require.NoError(t, proto.CompileLefthook(output))

	var buf bytes.Buffer

	require.NoError(t, output.GenerateFile("lefthook.yml", &buf))

	rendered := buf.String()
	assert.Contains(t, rendered, "name: generate frontend")
	assert.Contains(t, rendered, "run: make generate-frontend")
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
		if hook.Jobs[0].Group.Jobs[i].Name == "generate frontend" {
			generateJob = &hook.Jobs[0].Group.Jobs[i]

			break
		}
	}

	require.NotNil(t, generateJob, "generate frontend job not found")
	assert.Equal(t, "testorg", generateJob.Env["USERNAME"])
}

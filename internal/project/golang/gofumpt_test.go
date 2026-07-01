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
	"github.com/siderolabs/kres/internal/output/lefthook"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/golang"
	"github.com/siderolabs/kres/internal/project/meta"
)

func TestGofumptInterfaces(t *testing.T) {
	assert.Implements(t, (*dockerfile.Compiler)(nil), new(golang.Gofumpt))
	assert.Implements(t, (*makefile.Compiler)(nil), new(golang.Gofumpt))
	assert.Implements(t, (*lefthook.Compiler)(nil), new(golang.Gofumpt))
}

func TestGofumptLefthook(t *testing.T) {
	gofumpt := golang.NewGofumpt(&meta.Options{
		GitHubOrganization: "testorg",
	}, ".")

	output := lefthook.NewOutput()
	output.Enable()

	require.NoError(t, gofumpt.CompileLefthook(output))

	var buf bytes.Buffer

	require.NoError(t, output.GenerateFile("lefthook.yml", &buf))

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

	var fmtJob *struct {
		Env  map[string]string `yaml:"env"`
		Name string            `yaml:"name"`
	}

	for i := range hook.Jobs[0].Group.Jobs {
		if hook.Jobs[0].Group.Jobs[i].Name == "fmt" {
			fmtJob = &hook.Jobs[0].Group.Jobs[i]

			break
		}
	}

	require.NotNil(t, fmtJob, "fmt job not found")
	assert.Equal(t, "testorg", fmtJob.Env["USERNAME"])
}

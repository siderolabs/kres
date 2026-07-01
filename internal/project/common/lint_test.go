// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"

	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/output/lefthook"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/common"
	"github.com/siderolabs/kres/internal/project/meta"
)

func TestLintInterfaces(t *testing.T) {
	assert.Implements(t, (*makefile.Compiler)(nil), new(common.Lint))
	assert.Implements(t, (*ghworkflow.Compiler)(nil), new(common.Lint))
	assert.Implements(t, (*lefthook.Compiler)(nil), new(common.Lint))
}

func TestLintLefthook(t *testing.T) {
	lint := common.NewLint(&meta.Options{
		GitHubOrganization: "testorg",
	})

	output := lefthook.NewOutput()
	output.Enable()

	require.NoError(t, lint.CompileLefthook(output))

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
	require.NotEmpty(t, hook.Jobs, "should have at least one job group")

	// Check lint-fmt job in PreCommitFixStage
	require.NotNil(t, hook.Jobs[0].Group)

	var lintFmtJob *struct {
		Env  map[string]string `yaml:"env"`
		Name string            `yaml:"name"`
	}

	for i := range hook.Jobs[0].Group.Jobs {
		if hook.Jobs[0].Group.Jobs[i].Name == "lint-fmt" {
			lintFmtJob = &hook.Jobs[0].Group.Jobs[i]

			break
		}
	}

	require.NotNil(t, lintFmtJob, "lint-fmt job not found")
	assert.Equal(t, "testorg", lintFmtJob.Env["USERNAME"])

	// Check lint job in PreCommitLintStage
	require.Len(t, hook.Jobs, 2, "should have two job groups (fix and lint stages)")
	require.NotNil(t, hook.Jobs[1].Group)

	var lintJob *struct {
		Env  map[string]string `yaml:"env"`
		Name string            `yaml:"name"`
	}

	for i := range hook.Jobs[1].Group.Jobs {
		if hook.Jobs[1].Group.Jobs[i].Name == "lint" {
			lintJob = &hook.Jobs[1].Group.Jobs[i]

			break
		}
	}

	require.NotNil(t, lintJob, "lint job not found")
	assert.Equal(t, "testorg", lintJob.Env["USERNAME"])
}

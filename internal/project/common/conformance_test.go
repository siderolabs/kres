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

	"github.com/siderolabs/kres/internal/output/lefthook"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/common"
	"github.com/siderolabs/kres/internal/project/meta"
)

func TestConformanceInterfaces(t *testing.T) {
	assert.Implements(t, (*makefile.Compiler)(nil), new(common.Conformance))
	assert.Implements(t, (*lefthook.Compiler)(nil), new(common.Conformance))
}

func TestConformanceLefthook(t *testing.T) {
	conformance := common.NewConformance(&meta.Options{
		GitHubOrganization: "testorg",
	})

	output := lefthook.NewOutput()
	output.Enable()

	require.NoError(t, conformance.CompileLefthook(output))

	var buf bytes.Buffer

	require.NoError(t, output.GenerateFile("lefthook.yml", &buf))

	var decoded map[string]struct {
		Commands map[string]struct {
			Env map[string]string `yaml:"env"`
		} `yaml:"commands"`
	}

	require.NoError(t, yaml.Unmarshal(buf.Bytes(), &decoded))

	hook := decoded["post-commit"]
	require.NotEmpty(t, hook.Commands)

	conformanceCmd, exists := hook.Commands["conformance"]
	require.True(t, exists, "conformance command not found")
	assert.Equal(t, "testorg", conformanceCmd.Env["USERNAME"])
}

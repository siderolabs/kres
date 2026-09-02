// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"

	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/golangci"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/golang"
)

func TestGolangciLintInterfaces(t *testing.T) {
	assert.Implements(t, (*dockerfile.Compiler)(nil), new(golang.GolangciLint))
	assert.Implements(t, (*makefile.Compiler)(nil), new(golang.GolangciLint))
}

func configNode(t *testing.T, fragment string) yaml.Node {
	t.Helper()

	var node yaml.Node

	require.NoError(t, yaml.Unmarshal([]byte(fragment), &node))

	return *node.Content[0]
}

func TestGolangciLintCompileConfig(t *testing.T) {
	for _, test := range []struct {
		lint          *golang.GolangciLint
		name          string
		expectedError string
	}{
		{
			name: "config with build tags",
			lint: &golang.GolangciLint{BuildTags: []string{"integration"}, Config: configNode(t, "linters:\n  enable:\n    - forbidigo\n")},
		},
		{
			name:          "removed depguard rules field",
			lint:          &golang.GolangciLint{DepguardExtraRules: map[string]any{}},
			expectedError: "depguardExtraRules is no more supported",
		},
		{
			name:          "empty config",
			lint:          &golang.GolangciLint{Config: configNode(t, "null\n")},
			expectedError: "config must be a mapping",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.lint.CompileGolangci(golangci.NewOutput())

			if test.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, test.expectedError)
			}
		})
	}
}

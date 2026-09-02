// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golangci_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"

	"github.com/siderolabs/kres/internal/output/golangci"
)

// render generates the config with the given fragment and returns it without the preamble.
func render(t *testing.T, fragment string) string {
	t.Helper()

	out := golangci.NewOutput()
	out.Enable()

	if fragment != "" {
		var config yaml.Node

		require.NoError(t, yaml.Unmarshal([]byte(fragment), &config))

		out.NewFile(".", *config.Content[0])
	} else {
		out.NewFile(".")
	}

	var buf bytes.Buffer

	require.NoError(t, out.GenerateFile(".golangci.yml", &buf))

	var parsed map[string]any

	require.NoError(t, yaml.Unmarshal(buf.Bytes(), &parsed), "generated config must be valid YAML:\n%s", buf.String())

	_, body, found := strings.Cut(buf.String(), "\n\n")
	require.True(t, found, "preamble must be followed by an empty line")

	return body
}

// TestDefaults makes sure the base config comes out unchanged, so that it is in the canonical form of the encoder.
func TestDefaults(t *testing.T) {
	base, err := os.ReadFile("golangci.yml")
	require.NoError(t, err)

	assert.Equal(t, string(base), render(t, ""))
}

func TestMergeIntoExistingSettings(t *testing.T) {
	config := render(t, `
linters:
  settings:
    errcheck:
      exclude-functions:
        - fmt.Fprintf
        - github.com/org/repo/pkg/safeout.Fprintf
    forbidigo:
      forbid:
        - pattern: ^fmt\.Print.*$
          msg: print through the filtered output
`)

	assert.Equal(t, 1, strings.Count(config, "\n    errcheck:\n"), "the existing errcheck block must be extended, not duplicated")
	assert.Contains(t, config, `    errcheck:
      check-type-assertions: true
      check-blank: true
      exclude-functions:
        - fmt.Fprintf
        - github.com/org/repo/pkg/safeout.Fprintf
`)
	assert.Contains(t, config, `    forbidigo:
      forbid:
        - pattern: ^fmt\.Print.*$
          msg: print through the filtered output
`)
}

// TestEnableRemovesFromDisable pins the one rule beyond plain merging: golangci-lint silently keeps a linter
// disabled when it is listed in both lists, so enabling one must take it off the disabled list.
func TestEnableRemovesFromDisable(t *testing.T) {
	config := render(t, `
linters:
  enable:
    - forbidigo
`)

	assert.Contains(t, config, "  disable:\n    - exhaustruct\n    - err113\n    - funcorder\n", "forbidigo is gone from between err113 and funcorder, the rest stays disabled")
	assert.Contains(t, config, "  enable:\n    - forbidigo\n")
}

func TestAppendToLists(t *testing.T) {
	config := render(t, `
run:
  build-tags:
    - integration
linters:
  exclusions:
    rules:
      - linters:
          - forbidigo
        path-except: ^cmd/cli/
    paths:
      - vendor$
`)

	assert.Contains(t, config, "  build-tags:\n    - integration\n")
	assert.Contains(t, config, `  exclusions:
    generated: lax
    paths:
      - third_party$
      - builtin$
      - examples$
      - vendor$
    rules:
      - linters:
          - forbidigo
        path-except: ^cmd/cli/
`)
}

func TestReplaceScalar(t *testing.T) {
	config := render(t, `
linters:
  settings:
    lll:
      line-length: 120
`)

	assert.Contains(t, config, "    lll:\n      line-length: 120\n      tab-width: 4\n")
}

// TestFragmentsPerFile makes sure the fragments of one project don't leak into the config of another one.
func TestFragmentsPerFile(t *testing.T) {
	var config yaml.Node

	require.NoError(t, yaml.Unmarshal([]byte("linters:\n  enable:\n    - forbidigo\n"), &config))

	out := golangci.NewOutput()
	out.Enable()
	out.NewFile(".", *config.Content[0])
	out.NewFile("client")

	var root, client bytes.Buffer

	require.NoError(t, out.GenerateFile(".golangci.yml", &root))
	require.NoError(t, out.GenerateFile("client/.golangci.yml", &client))

	assert.Contains(t, root.String(), "  enable:\n    - forbidigo\n")
	assert.NotContains(t, client.String(), "  enable:\n    - forbidigo\n")
	assert.Contains(t, client.String(), "\n    - forbidigo\n", "still disabled in the other project")
}

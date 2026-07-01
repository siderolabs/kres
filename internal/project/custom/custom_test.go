// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package custom_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"

	"github.com/siderolabs/kres/internal/output/lefthook"
	"github.com/siderolabs/kres/internal/project/custom"
	"github.com/siderolabs/kres/internal/project/meta"
)

func TestCompileLefthookDisabled(t *testing.T) {
	step := custom.NewStep(&meta.Options{}, "custom")
	step.Lefthook.Hooks = map[string]*lefthook.Hook{
		"pre-commit": {
			Jobs: []*lefthook.Job{{Name: "custom", Run: "make custom"}},
		},
	}

	o := lefthook.NewOutput()
	o.Enable()

	require.NoError(t, step.CompileLefthook(o))

	// Lefthook is disabled, so the step contributes no jobs.
	assert.Nil(t, o.Hook("pre-commit").Jobs)
}

// lefthookFile mirrors the structure of a generated lefthook.yml closely enough
// to assert on hooks, commands, jobs and nested groups.
type lefthookFile map[string]struct { //nolint:govet
	Parallel *bool `yaml:"parallel"`
	Commands map[string]struct {
		Run string `yaml:"run"`
	} `yaml:"commands"`
	Jobs []lefthookJob `yaml:"jobs"`
}

type lefthookJob struct { //nolint:govet
	Name       string `yaml:"name"`
	Run        string `yaml:"run"`
	StageFixed bool   `yaml:"stage_fixed"`
	Group      *struct {
		Parallel *bool         `yaml:"parallel"`
		Jobs     []lefthookJob `yaml:"jobs"`
	} `yaml:"group"`
}

func decodeLefthook(t *testing.T, step *custom.Step) lefthookFile {
	t.Helper()

	o := lefthook.NewOutput()
	o.Enable()

	require.NoError(t, step.CompileLefthook(o))

	var buf bytes.Buffer

	require.NoError(t, o.GenerateFile("lefthook.yml", &buf))

	var decoded lefthookFile

	require.NoError(t, yaml.Unmarshal(buf.Bytes(), &decoded))

	return decoded
}

// TestCompileLefthookJobsWithGroups covers the pre-commit shape from the
// generated lefthook.yml: top-level named jobs each wrapping a sequential group
// of stage-fixed jobs.
func TestCompileLefthookJobsWithGroups(t *testing.T) {
	step := custom.NewStep(&meta.Options{}, "custom")
	step.Lefthook.Enabled = true
	step.Lefthook.Hooks = map[string]*lefthook.Hook{
		"pre-commit": {
			Jobs: []*lefthook.Job{
				{
					Name: lefthook.PreCommitFixStage,
					Group: &lefthook.Group{
						Parallel: new(false),
						Jobs: []*lefthook.Job{
							{Name: "generate", Run: "make generate", StageFixed: true},
							{Name: "fmt", Run: "make fmt", StageFixed: true},
						},
					},
				},
				{
					Name: lefthook.PreCommitLintStage,
					Group: &lefthook.Group{
						Parallel: new(false),
						Jobs: []*lefthook.Job{
							{Name: "lint", Run: "make lint"},
						},
					},
				},
			},
		},
	}

	hook := decodeLefthook(t, step)["pre-commit"]

	require.Len(t, hook.Jobs, 2)

	fix := hook.Jobs[0]
	assert.Equal(t, lefthook.PreCommitFixStage, fix.Name)
	require.NotNil(t, fix.Group)
	require.NotNil(t, fix.Group.Parallel)
	assert.False(t, *fix.Group.Parallel)
	require.Len(t, fix.Group.Jobs, 2)
	assert.Equal(t, "generate", fix.Group.Jobs[0].Name)
	assert.Equal(t, "make generate", fix.Group.Jobs[0].Run)
	assert.True(t, fix.Group.Jobs[0].StageFixed)
	assert.Equal(t, "fmt", fix.Group.Jobs[1].Name)
	assert.True(t, fix.Group.Jobs[1].StageFixed)

	lint := hook.Jobs[1]
	assert.Equal(t, lefthook.PreCommitLintStage, lint.Name)
	require.NotNil(t, lint.Group)
	require.Len(t, lint.Group.Jobs, 1)
	assert.Equal(t, "lint", lint.Group.Jobs[0].Name)
	assert.False(t, lint.Group.Jobs[0].StageFixed)
}

// TestCompileLefthookCommands covers the commit-msg shape: a commands-style
// hook with an explicit parallel: false.
func TestCompileLefthookCommands(t *testing.T) {
	step := custom.NewStep(&meta.Options{}, "custom")
	step.Lefthook.Enabled = true
	step.Lefthook.Hooks = map[string]*lefthook.Hook{
		"post-commit": {
			Parallel: new(false),
			Commands: map[string]*lefthook.Command{
				"conformance": {Run: "make conformance"},
			},
		},
	}

	hook := decodeLefthook(t, step)["post-commit"]

	require.NotNil(t, hook.Parallel)
	assert.False(t, *hook.Parallel)
	require.Contains(t, hook.Commands, "conformance")
	assert.Equal(t, "make conformance", hook.Commands["conformance"].Run)
}

// TestCompileLefthookMergesNamedGroup verifies a custom step appends to an
// existing named group rather than forking a new one.
func TestCompileLefthookMergesNamedGroup(t *testing.T) {
	step := custom.NewStep(&meta.Options{}, "custom")
	step.Lefthook.Enabled = true
	step.Lefthook.Hooks = map[string]*lefthook.Hook{
		"pre-commit": {
			Jobs: []*lefthook.Job{
				{
					Name: lefthook.PreCommitFixStage,
					Group: &lefthook.Group{
						Jobs: []*lefthook.Job{{Name: "custom", Run: "make custom", StageFixed: true}},
					},
				},
			},
		},
	}

	o := lefthook.NewOutput()
	o.Enable()

	// Pre-seed the shared fix group, as a standard block would.
	o.Hook("pre-commit").Group(lefthook.PreCommitFixStage).
		WithParallel(false).
		Job().WithName("generate").WithRun("make generate").WithStageFixed()

	require.NoError(t, step.CompileLefthook(o))

	var buf bytes.Buffer

	require.NoError(t, o.GenerateFile("lefthook.yml", &buf))

	var decoded lefthookFile

	require.NoError(t, yaml.Unmarshal(buf.Bytes(), &decoded))

	hook := decoded["pre-commit"]
	require.Len(t, hook.Jobs, 1, "custom step must extend the existing fix group, not add a new top-level job")

	group := hook.Jobs[0].Group
	require.NotNil(t, group)
	require.Len(t, group.Jobs, 2)
	assert.Equal(t, "generate", group.Jobs[0].Name)
	assert.Equal(t, "custom", group.Jobs[1].Name)
}

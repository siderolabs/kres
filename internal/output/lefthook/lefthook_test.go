// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lefthook_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"

	"github.com/siderolabs/kres/internal/output/lefthook"
)

func TestDisabledOutputProducesNoFiles(t *testing.T) {
	o := lefthook.NewOutput()

	assert.Nil(t, o.Filenames())
}

func TestCommandsRoundTrip(t *testing.T) {
	o := lefthook.NewOutput()
	o.Enable()

	preCommit := o.Hook("pre-commit")
	preCommit.Command("fmt").WithRun("make fmt")
	preCommit.Command("lint").WithRun("make lint")

	o.Hook("commit-msg").Command("conformance").WithRun("make conformance")

	require.Equal(t, []string{"lefthook.yml"}, o.Filenames())

	var buf bytes.Buffer

	require.NoError(t, o.GenerateFile("lefthook.yml", &buf))

	var decoded map[string]struct {
		Commands map[string]struct {
			Run string `yaml:"run"`
		} `yaml:"commands"`
	}

	require.NoError(t, yaml.Unmarshal(buf.Bytes(), &decoded))

	require.Contains(t, decoded, "pre-commit")
	require.Contains(t, decoded, "commit-msg")

	assert.Equal(t, "make fmt", decoded["pre-commit"].Commands["fmt"].Run)
	assert.Equal(t, "make lint", decoded["pre-commit"].Commands["lint"].Run)
	assert.Equal(t, "make conformance", decoded["commit-msg"].Commands["conformance"].Run)
}

func TestParallelDefaultIsOmitted(t *testing.T) {
	o := lefthook.NewOutput()
	o.Enable()

	o.Hook("pre-commit").Command("fmt").WithRun("make fmt")

	var buf bytes.Buffer

	require.NoError(t, o.GenerateFile("lefthook.yml", &buf))

	assert.NotContains(t, buf.String(), "parallel:")
}

func TestParallelFalseIsEmitted(t *testing.T) {
	o := lefthook.NewOutput()
	o.Enable()

	o.Hook("commit-msg").
		WithParallel(false).
		Command("conformance").WithRun("make conformance")

	var buf bytes.Buffer

	require.NoError(t, o.GenerateFile("lefthook.yml", &buf))

	// *bool with explicit false must emit; bool+omitempty would have swallowed it.
	assert.Contains(t, buf.String(), "parallel: false")
}

func TestJobsAndGroups(t *testing.T) {
	o := lefthook.NewOutput()
	o.Enable()

	preCommit := o.Hook("pre-commit")

	parallelGroup := preCommit.Job().AsGroup().WithParallel(true)
	parallelGroup.Job().WithRun("make fmt")
	parallelGroup.Job().WithRun("make generate")

	sequentialGroup := preCommit.Job().AsGroup().WithParallel(false)
	sequentialGroup.Job().WithRun("make lint")

	var buf bytes.Buffer

	require.NoError(t, o.GenerateFile("lefthook.yml", &buf))

	var decoded map[string]struct {
		Jobs []struct {
			Group *struct { //nolint:govet
				Parallel *bool `yaml:"parallel"`
				Jobs     []struct {
					Run string `yaml:"run"`
				} `yaml:"jobs"`
			} `yaml:"group"`
		} `yaml:"jobs"`
	}

	require.NoError(t, yaml.Unmarshal(buf.Bytes(), &decoded))

	hook := decoded["pre-commit"]
	require.Len(t, hook.Jobs, 2)

	require.NotNil(t, hook.Jobs[0].Group)
	require.NotNil(t, hook.Jobs[0].Group.Parallel)
	assert.True(t, *hook.Jobs[0].Group.Parallel)
	require.Len(t, hook.Jobs[0].Group.Jobs, 2)
	assert.Equal(t, "make fmt", hook.Jobs[0].Group.Jobs[0].Run)
	assert.Equal(t, "make generate", hook.Jobs[0].Group.Jobs[1].Run)

	require.NotNil(t, hook.Jobs[1].Group)
	require.NotNil(t, hook.Jobs[1].Group.Parallel)
	assert.False(t, *hook.Jobs[1].Group.Parallel)
	require.Len(t, hook.Jobs[1].Group.Jobs, 1)
	assert.Equal(t, "make lint", hook.Jobs[1].Group.Jobs[0].Run)
}

func TestNamedGroupsAreShared(t *testing.T) {
	o := lefthook.NewOutput()
	o.Enable()

	preCommit := o.Hook("pre-commit")

	// Two blocks append to the same stage-1 group via the same key.
	preCommit.Group(lefthook.PreCommitFixStage).WithParallel(false).Job().WithName("lint-fmt").WithRun("make lint-fmt").WithStageFixed()
	preCommit.Group(lefthook.PreCommitFixStage).Job().WithName("generate").WithRun("make generate").WithStageFixed()

	// A distinct key creates a second, ordered-after group.
	preCommit.Group(lefthook.PreCommitLintStage).WithParallel(false).Job().WithRun("make lint")

	var buf bytes.Buffer

	require.NoError(t, o.GenerateFile("lefthook.yml", &buf))

	var decoded map[string]struct {
		Jobs []struct { //nolint:govet
			Name  string    `yaml:"name"`
			Group *struct { //nolint:govet
				Parallel *bool `yaml:"parallel"`
				Jobs     []struct {
					Name       string `yaml:"name"`
					Run        string `yaml:"run"`
					StageFixed bool   `yaml:"stage_fixed"`
				} `yaml:"jobs"`
			} `yaml:"group"`
		} `yaml:"jobs"`
	}

	require.NoError(t, yaml.Unmarshal(buf.Bytes(), &decoded))

	hook := decoded["pre-commit"]
	require.Len(t, hook.Jobs, 2, "same key must reuse one group, distinct key adds a second")

	// The group key is emitted as the wrapping job's name.
	assert.Equal(t, lefthook.PreCommitFixStage, hook.Jobs[0].Name)
	require.NotNil(t, hook.Jobs[0].Group)
	require.Len(t, hook.Jobs[0].Group.Jobs, 2)
	assert.Equal(t, "lint-fmt", hook.Jobs[0].Group.Jobs[0].Name)
	assert.Equal(t, "make lint-fmt", hook.Jobs[0].Group.Jobs[0].Run)
	assert.True(t, hook.Jobs[0].Group.Jobs[0].StageFixed)
	assert.Equal(t, "generate", hook.Jobs[0].Group.Jobs[1].Name)
	assert.Equal(t, "make generate", hook.Jobs[0].Group.Jobs[1].Run)

	assert.Equal(t, lefthook.PreCommitLintStage, hook.Jobs[1].Name)
	require.NotNil(t, hook.Jobs[1].Group)
	require.Len(t, hook.Jobs[1].Group.Jobs, 1)
	assert.Equal(t, "make lint", hook.Jobs[1].Group.Jobs[0].Run)
}

func TestJobBuilders(t *testing.T) {
	o := lefthook.NewOutput()
	o.Enable()

	o.Hook("pre-commit").
		Job().
		WithName("docs-check").
		WithRun("echo $E1").
		WithEnv("E1", "hello").
		WithEnv("E2", "world").
		WithGlob("*.md").
		WithExclude("README.md").
		WithRoot("subdir/").
		WithTags("docs").
		WithSkip("merge").
		WithOnly("ref: main").
		WithInteractive().
		WithStageFixed().
		WithPriority(5)

	var buf bytes.Buffer

	require.NoError(t, o.GenerateFile("lefthook.yml", &buf))

	var decoded map[string]struct {
		Jobs []struct { //nolint:govet
			Name        string            `yaml:"name"`
			Run         string            `yaml:"run"`
			Env         map[string]string `yaml:"env"`
			Glob        []string          `yaml:"glob"`
			Exclude     []string          `yaml:"exclude"`
			Root        string            `yaml:"root"`
			Tags        []string          `yaml:"tags"`
			Skip        []string          `yaml:"skip"`
			Only        []string          `yaml:"only"`
			Interactive bool              `yaml:"interactive"`
			StageFixed  bool              `yaml:"stage_fixed"`
			Priority    int               `yaml:"priority"`
		} `yaml:"jobs"`
	}

	require.NoError(t, yaml.Unmarshal(buf.Bytes(), &decoded))

	require.Len(t, decoded["pre-commit"].Jobs, 1)

	job := decoded["pre-commit"].Jobs[0]
	assert.Equal(t, "docs-check", job.Name)
	assert.Equal(t, "echo $E1", job.Run)
	assert.Equal(t, map[string]string{"E1": "hello", "E2": "world"}, job.Env)
	assert.Equal(t, []string{"*.md"}, job.Glob)
	assert.Equal(t, []string{"README.md"}, job.Exclude)
	assert.Equal(t, "subdir/", job.Root)
	assert.Equal(t, []string{"docs"}, job.Tags)
	assert.Equal(t, []string{"merge"}, job.Skip)
	assert.Equal(t, []string{"ref: main"}, job.Only)
	assert.True(t, job.Interactive)
	assert.True(t, job.StageFixed)
	assert.Equal(t, 5, job.Priority)
}

func TestCommandBuilders(t *testing.T) {
	o := lefthook.NewOutput()
	o.Enable()

	o.Hook("pre-commit").
		WithParallel(true).
		Command("go-fmt").
		WithRun("gofmt -l -w {staged_files}").
		WithTags("format", "go").
		WithGlob("*.go").
		WithFiles("git diff --name-only").
		WithSkip("merge", "rebase").
		WithOnly("ref: main").
		WithInteractive().
		WithStageFixed().
		WithPriority(10)

	var buf bytes.Buffer

	require.NoError(t, o.GenerateFile("lefthook.yml", &buf))

	var decoded map[string]struct { //nolint:govet
		Parallel *bool               `yaml:"parallel"`
		Commands map[string]struct { //nolint:govet
			Run         string   `yaml:"run"`
			Tags        []string `yaml:"tags"`
			Glob        string   `yaml:"glob"`
			Files       string   `yaml:"files"`
			Skip        []string `yaml:"skip"`
			Only        []string `yaml:"only"`
			Interactive bool     `yaml:"interactive"`
			StageFixed  bool     `yaml:"stage_fixed"`
			Priority    int      `yaml:"priority"`
		} `yaml:"commands"`
	}

	require.NoError(t, yaml.Unmarshal(buf.Bytes(), &decoded))

	hook := decoded["pre-commit"]
	require.NotNil(t, hook.Parallel)
	assert.True(t, *hook.Parallel)

	cmd := hook.Commands["go-fmt"]
	assert.Equal(t, "gofmt -l -w {staged_files}", cmd.Run)
	assert.Equal(t, []string{"format", "go"}, cmd.Tags)
	assert.Equal(t, "*.go", cmd.Glob)
	assert.Equal(t, "git diff --name-only", cmd.Files)
	assert.Equal(t, []string{"merge", "rebase"}, cmd.Skip)
	assert.Equal(t, []string{"ref: main"}, cmd.Only)
	assert.True(t, cmd.Interactive)
	assert.True(t, cmd.StageFixed)
	assert.Equal(t, 10, cmd.Priority)
}

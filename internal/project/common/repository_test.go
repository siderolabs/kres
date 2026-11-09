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

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/project/common"
	"github.com/siderolabs/kres/internal/project/meta"
)

const conformWorkflow = ".github/workflows/conform.yaml"

func TestRepositoryInterfaces(t *testing.T) {
	assert.Implements(t, (*ghworkflow.Compiler)(nil), new(common.Repository))
}

func TestRepositoryConformWorkflow(t *testing.T) {
	repo := common.NewRepository(&meta.Options{
		MainBranch: "main",
	})

	o := ghworkflow.NewOutput("main", true, false, "")

	require.NoError(t, repo.CompileGitHubWorkflow(o))
	require.Contains(t, o.Filenames(), conformWorkflow)

	var buf bytes.Buffer

	require.NoError(t, o.GenerateFile(conformWorkflow, &buf))

	var workflow struct {
		Permissions map[string]string `yaml:"permissions"`
		Jobs        map[string]struct {
			Steps []struct {
				With map[string]string `yaml:"with"`
				Env  map[string]string `yaml:"env"`
				Run  string            `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
		Name        string `yaml:"name"`
		Concurrency struct {
			Group string `yaml:"group"`
		} `yaml:"concurrency"`
		On struct {
			Push        *struct{} `yaml:"push"`
			PullRequest struct {
				Branches []string `yaml:"branches"`
			} `yaml:"pull_request"`
		} `yaml:"on"`
	}

	require.NoError(t, yaml.Unmarshal(buf.Bytes(), &workflow))

	assert.Equal(t, "conform", workflow.Name)
	// keyed by PR number: fork PRs from branches with the same name must not cancel each other
	assert.Equal(t, "${{ github.workflow }}-${{ github.event.pull_request.number || github.run_id }}", workflow.Concurrency.Group)
	assert.Equal(t, []string{"main", "release-*"}, workflow.On.PullRequest.Branches)
	assert.Nil(t, workflow.On.Push, "conform should run only on pull requests")
	assert.Equal(t, map[string]string{"contents": "read"}, workflow.Permissions)

	require.Contains(t, workflow.Jobs, "conform")

	steps := workflow.Jobs["conform"].Steps
	require.Len(t, steps, 3)

	assert.Equal(t, "${{ github.event.pull_request.head.sha }}", steps[0].With["ref"])
	assert.Equal(t, "0", steps[0].With["fetch-depth"])
	assert.Equal(t, "false", steps[0].With["persist-credentials"])

	assert.Equal(t, config.ConformVersion, steps[1].Env["CONFORM_VERSION"])
	assert.Contains(t, steps[1].Run, "sha256sum -c -")

	assert.Equal(t, "${{ github.base_ref }}", steps[2].Env["BASE_REF"])
	assert.Equal(t, "${{ secrets.GITHUB_TOKEN }}", steps[2].Env["GITHUB_TOKEN"])
	assert.NotContains(t, steps[2].Run, "${{", "expressions must not be interpolated into the script")
	assert.Contains(t, steps[2].Run, `--base-branch="origin/${BASE_REF}"`)
	assert.Contains(t, steps[2].Run, `--commit-ref="refs/remotes/origin/${BASE_REF}"`)
}

func TestRepositoryConformWorkflowDisabled(t *testing.T) {
	repo := common.NewRepository(&meta.Options{
		MainBranch: "main",
	})
	repo.EnableConform = false

	o := ghworkflow.NewOutput("main", true, false, "")

	require.NoError(t, repo.CompileGitHubWorkflow(o))
	assert.NotContains(t, o.Filenames(), conformWorkflow)
}

const slashMergeWorkflow = ".github/workflows/slash-merge.yaml"

func TestRepositorySlashMergeWorkflow(t *testing.T) {
	repo := common.NewRepository(&meta.Options{
		MainBranch: "main",
	})
	repo.SlashMerge = true

	o := ghworkflow.NewOutput("main", true, false, "")

	require.NoError(t, repo.CompileGitHubWorkflow(o))
	require.Contains(t, o.Filenames(), slashMergeWorkflow)

	var buf bytes.Buffer

	require.NoError(t, o.GenerateFile(slashMergeWorkflow, &buf))

	var workflow struct {
		Permissions map[string]string `yaml:"permissions"`
		Jobs        map[string]struct {
			Permissions map[string]string `yaml:"permissions"`
			RunsOn      map[string]string `yaml:"runs-on"`
			Environment struct {
				Name string `yaml:"name"`
			} `yaml:"environment"`
			If    string `yaml:"if"`
			Steps []struct {
				With map[string]string `yaml:"with"`
				Env  map[string]string `yaml:"env"`
				Name string            `yaml:"name"`
				If   string            `yaml:"if"`
				Run  string            `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
		On struct {
			IssueComment struct {
				Types []string `yaml:"types"`
			} `yaml:"issue_comment"`
		} `yaml:"on"`
	}

	require.NoError(t, yaml.Unmarshal(buf.Bytes(), &workflow))

	assert.Equal(t, []string{"created"}, workflow.On.IssueComment.Types)
	assert.NotNil(t, workflow.Permissions)
	assert.Empty(t, workflow.Permissions, "workflow-level permissions must deny everything")

	require.Contains(t, workflow.Jobs, "slash-merge")

	job := workflow.Jobs["slash-merge"]

	assert.Equal(t, map[string]string{"group": ghworkflow.GenericRunner}, job.RunsOn)
	assert.Contains(t, job.If, "github.event.issue.pull_request")
	assert.Equal(t, "read", job.Permissions["contents"], "GITHUB_TOKEN must not be able to push")
	assert.Equal(t, "slash-merge", job.Environment.Name, "push token must be an environment secret")

	var appToken map[string]string

	for _, step := range job.Steps {
		if step.Name == "app token" {
			appToken = step.With
		}
	}

	require.NotNil(t, appToken)
	assert.Equal(t, "${{ vars.SLASH_MERGE_APP_CLIENT_ID }}", appToken["client-id"])
	assert.Equal(t, "${{ secrets.SLASH_MERGE_APP_PRIVATE_KEY }}", appToken["private-key"])
	assert.Equal(t, "write", appToken["permission-contents"])
	assert.Equal(t, "write", appToken["permission-workflows"])
	assert.NotContains(t, appToken, "owner", "the token must be scoped to the current repository")
	assert.NotContains(t, appToken, "repositories", "the token must be scoped to the current repository")

	for _, step := range job.Steps {
		assert.NotContains(t, step.With["script"], "${{", "expressions must not be interpolated into scripts: step %q", step.Name)
		assert.NotContains(t, step.Run, "${{", "expressions must not be interpolated into scripts: step %q", step.Name)

		if step.With["private-key"] != "" || step.With["token"] != "" {
			assert.Equal(t, "steps.validate.outputs.sha != ''", step.If, "push token must be created and used only after validation: step %q", step.Name)
		}

		for _, value := range step.With {
			assert.NotContains(t, value, "secrets.GITHUB_TOKEN", "step %q", step.Name)
		}
	}
}

func TestRepositorySlashMergeWorkflowDisabled(t *testing.T) {
	repo := common.NewRepository(&meta.Options{
		MainBranch: "main",
	})

	o := ghworkflow.NewOutput("main", true, false, "")

	require.NoError(t, repo.CompileGitHubWorkflow(o))
	assert.NotContains(t, o.Filenames(), slashMergeWorkflow)
}

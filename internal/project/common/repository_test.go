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

const skipBotPullRequests = "github.event.pull_request.user.login != 'dependabot[bot]' && github.event.pull_request.user.login != 'renovate[bot]'"

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
			If    string `yaml:"if"`
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

	assert.Equal(t, skipBotPullRequests, workflow.Jobs["conform"].If)

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

const projectAutoAddWorkflow = ".github/workflows/project-auto-add.yaml"

func TestRepositoryProjectAutoAddWorkflowDisabled(t *testing.T) {
	repo := common.NewRepository(&meta.Options{
		MainBranch:         "main",
		GitHubOrganization: "siderolabs",
	})

	require.NoError(t, yaml.Unmarshal([]byte("projectAutoAdd:\n  enabled: false\n"), repo))

	o := ghworkflow.NewOutput("main", true, false, "")

	require.NoError(t, repo.CompileGitHubWorkflow(o))
	assert.NotContains(t, o.Filenames(), projectAutoAddWorkflow)
}

func TestRepositoryProjectAutoAddWorkflowNotGitHub(t *testing.T) {
	// the organization is not detected for non-GitHub repositories
	repo := common.NewRepository(&meta.Options{
		MainBranch: "main",
	})

	o := ghworkflow.NewOutput("main", true, false, "")

	require.NoError(t, repo.CompileGitHubWorkflow(o))
	assert.NotContains(t, o.Filenames(), projectAutoAddWorkflow)
}

func TestRepositoryProjectAutoAddWorkflow(t *testing.T) {
	repo := common.NewRepository(&meta.Options{
		MainBranch:         "main",
		GitHubOrganization: "siderolabs",
	})

	// a partial config keeps the defaults for the fields which are not set
	require.NoError(t, yaml.Unmarshal([]byte("projectAutoAdd:\n  readyStatus: Review\n"), repo))

	o := ghworkflow.NewOutput("main", true, false, "")

	require.NoError(t, repo.CompileGitHubWorkflow(o))
	require.Contains(t, o.Filenames(), projectAutoAddWorkflow)

	var buf bytes.Buffer

	require.NoError(t, o.GenerateFile(projectAutoAddWorkflow, &buf))

	var workflow struct {
		Permissions map[string]string `yaml:"permissions"`
		Jobs        map[string]struct {
			If    string `yaml:"if"`
			Steps []struct {
				With map[string]string `yaml:"with"`
				Env  map[string]string `yaml:"env"`
				ID   string            `yaml:"id"`
				Uses string            `yaml:"uses"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
		Name        string `yaml:"name"`
		Concurrency struct {
			Group string `yaml:"group"`
		} `yaml:"concurrency"`
		On struct {
			Push              *struct{} `yaml:"push"`
			PullRequest       *struct{} `yaml:"pull_request"`
			PullRequestTarget struct {
				Types []string `yaml:"types"`
			} `yaml:"pull_request_target"`
		} `yaml:"on"`
	}

	require.NoError(t, yaml.Unmarshal(buf.Bytes(), &workflow))

	assert.Equal(t, "project-auto-add", workflow.Name)
	assert.Equal(t, "${{ github.workflow }}-${{ github.event.pull_request.number }}", workflow.Concurrency.Group)
	assert.Nil(t, workflow.On.Push)
	assert.Nil(t, workflow.On.PullRequest)
	assert.Equal(t, []string{"opened", "reopened", "ready_for_review", "converted_to_draft"}, workflow.On.PullRequestTarget.Types)
	assert.NotNil(t, workflow.Permissions)
	assert.Empty(t, workflow.Permissions, "GITHUB_TOKEN should have no permissions")

	require.Contains(t, workflow.Jobs, "project-auto-add")

	assert.Equal(t, skipBotPullRequests, workflow.Jobs["project-auto-add"].If)

	steps := workflow.Jobs["project-auto-add"].Steps
	require.Len(t, steps, 2)

	assert.Equal(t, "actions/create-github-app-token@"+config.CreateGitHubAppTokenActionRef, steps[0].Uses)
	assert.Equal(t, "app-token", steps[0].ID)
	assert.Equal(t, map[string]string{
		"client-id":                        "${{ secrets.PROJECT_APP_CLIENT_ID }}",
		"private-key":                      "${{ secrets.PROJECT_APP_PRIVATE_KEY }}",
		"owner":                            "siderolabs",
		"repositories":                     "${{ github.event.repository.name }}",
		"permission-organization-projects": "write",
		"permission-pull-requests":         "read",
	}, steps[0].With)

	assert.Equal(t, "actions/github-script@"+config.GitHubScriptActionRef, steps[1].Uses)
	assert.Equal(t, "${{ steps.app-token.outputs.token }}", steps[1].With["github-token"])
	assert.NotContains(t, steps[1].With["script"], "${{", "expressions must not be interpolated into the script")
	assert.Equal(t, map[string]string{
		"PROJECT_OWNER":           "siderolabs",
		"PROJECT_NUMBER":          "17",
		"PROJECT_STATUS_FIELD":    "Status",
		"PROJECT_DRAFT_STATUS":    "In Progress",
		"PROJECT_READY_STATUS":    "Review",
		"PROJECT_ITERATION_FIELD": "Iteration",
	}, steps[1].Env)
}

func TestRepositoryProjectAutoAddWorkflowOtherOrganization(t *testing.T) {
	repo := common.NewRepository(&meta.Options{
		MainBranch:         "main",
		GitHubOrganization: "siderolabs",
	})

	repo.ProjectAutoAdd.Organization = "other-org"

	o := ghworkflow.NewOutput("main", true, false, "")

	require.NoError(t, repo.CompileGitHubWorkflow(o))

	var buf bytes.Buffer

	require.NoError(t, o.GenerateFile(projectAutoAddWorkflow, &buf))

	var workflow struct {
		Jobs map[string]struct {
			Steps []struct {
				With map[string]string `yaml:"with"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}

	require.NoError(t, yaml.Unmarshal(buf.Bytes(), &workflow))

	steps := workflow.Jobs["project-auto-add"].Steps
	require.Len(t, steps, 2)

	// the installation in the other organization has no access to this repository, so the token can't be scoped to it
	assert.Equal(t, "other-org", steps[0].With["owner"])
	assert.NotContains(t, steps[0].With, "repositories")
}

func TestRepositoryProjectAutoAddWorkflowInvalid(t *testing.T) {
	for _, tc := range []struct {
		name    string
		modify  func(*common.ProjectAutoAdd)
		wantErr string
	}{
		{
			name:    "invalid secret name",
			modify:  func(cfg *common.ProjectAutoAdd) { cfg.AppPrivateKeySecret = "KEY }} ${{ github.token" },
			wantErr: `projectAutoAdd: invalid appPrivateKeySecret "KEY }} ${{ github.token"`,
		},
		{
			name:    "reserved secret name",
			modify:  func(cfg *common.ProjectAutoAdd) { cfg.AppClientIDSecret = "GITHUB_APP_ID" },
			wantErr: `projectAutoAdd: invalid appClientIDSecret "GITHUB_APP_ID"`,
		},
		{
			name:    "invalid organization",
			modify:  func(cfg *common.ProjectAutoAdd) { cfg.Organization = "${{ github.actor }}" },
			wantErr: `projectAutoAdd: invalid organization "${{ github.actor }}"`,
		},
		{
			name:    "expression in status",
			modify:  func(cfg *common.ProjectAutoAdd) { cfg.ReadyStatus = "${{ secrets.FOO }}" },
			wantErr: `projectAutoAdd: readyStatus "${{ secrets.FOO }}" should not contain expressions`,
		},
		{
			name:    "invalid project number",
			modify:  func(cfg *common.ProjectAutoAdd) { cfg.ProjectNumber = 0 },
			wantErr: "projectAutoAdd: invalid project number 0",
		},
		{
			name:    "empty status",
			modify:  func(cfg *common.ProjectAutoAdd) { cfg.DraftStatus = "" },
			wantErr: "projectAutoAdd: draftStatus is not set",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := common.NewRepository(&meta.Options{
				MainBranch:         "main",
				GitHubOrganization: "siderolabs",
			})

			tc.modify(&repo.ProjectAutoAdd)

			require.EqualError(t, repo.CompileGitHubWorkflow(ghworkflow.NewOutput("main", true, false, "")), tc.wantErr)
		})
	}
}

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"

	"github.com/google/go-github/v92/github"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
)

const (
	// slashMergeWorkflowName is the name of the slash-merge workflow and its job.
	slashMergeWorkflowName = "slash-merge"

	// slashMergeEnvironment is the deployment environment of the slash-merge job, which holds the GitHub App credentials.
	//
	// The environment only allows deployments from the main branch: issue_comment workflows always run
	// from the default branch, so the slash-merge job can access the secret, but workflows from other branches
	// (which can be pushed without any review by any user with write access) can't.
	slashMergeEnvironment = "slash-merge"

	// The push to the base branch is done with a GitHub App installation token, created for each run
	// and scoped to the current repository.
	//
	// GITHUB_TOKEN can't be used for the push: GitHub rejects pushes made with it which update
	// files under .github/workflows (there is no `workflows` permission for GITHUB_TOKEN), and
	// pushes made with GITHUB_TOKEN don't trigger any workflows, so CI wouldn't run on the base branch.

	// slashMergeAppClientIDVariable is the environment variable holding the GitHub App client ID.
	slashMergeAppClientIDVariable = "SLASH_MERGE_APP_CLIENT_ID"
	// slashMergeAppPrivateKeySecret is the environment secret holding the GitHub App private key.
	slashMergeAppPrivateKeySecret = "SLASH_MERGE_APP_PRIVATE_KEY"
)

// slashMergeValidateScript is the actions/github-script script which validates a slash-merge request.
//
//go:embed resources/slash-merge.js
var slashMergeValidateScript string

// slashMergeFastForwardScript pushes the validated PR head to the base branch.
//
// The push is not forced, so GitHub rejects it if it's not a fast-forward, or if the PR
// doesn't satisfy the branch protection rules of the base branch.
const slashMergeFastForwardScript = `set -eu

# the PR head must not have changed since validation
git fetch --no-tags origin "+refs/pull/${PR_NUMBER}/head:refs/remotes/origin/slash-merge-pr-head"

if [ "$(git rev-parse refs/remotes/origin/slash-merge-pr-head)" != "${PR_SHA}" ]; then
  echo "::error::PR head has changed since validation, please run /nm again"
  exit 1
fi

if ! git merge-base --is-ancestor "refs/remotes/origin/${BASE}" "${PR_SHA}"; then
  echo "::error::${PR_SHA} is not a fast-forward of ${BASE}, please rebase the PR"
  exit 1
fi

git push origin "${PR_SHA}:refs/heads/${BASE}"`

// slashMergeReportPushFailureScript comments on the PR when the push fails.
const slashMergeReportPushFailureScript = `const { owner, repo } = context.repo;
const runURL = ` + "`${context.serverUrl}/${owner}/${repo}/actions/runs/${context.runId}`" + `;

await github.rest.issues.createComment({
  owner,
  repo,
  issue_number: context.payload.issue.number,
  body: ` + "`⛔ @${context.payload.comment.user.login} failed to fast-forward the base branch, see [the workflow run](${runURL}) for details.`" + `,
});`

// slashMergeReactScript adds a reaction (passed via REACTION env var) to the triggering comment.
const slashMergeReactScript = `await github.rest.reactions.createForIssueComment({
  owner: context.repo.owner,
  repo: context.repo.repo,
  comment_id: context.payload.comment.id,
  content: process.env.REACTION,
});`

// slashMergeWorkflow builds the slash-merge workflow.
//
// A `/nm` comment on a PR by a user with write access to the repository fast-forwards
// the base branch to the PR head commit (so that commit SHAs and signatures are preserved).
//
// Security considerations:
//
//   - issue_comment workflows always run the workflow definition from the default branch,
//     and no PR code is executed or checked out, only the PR commits are fetched.
//   - The commenter must have write (or higher) permission on the repository, checked via the API
//     (author_association is not reliable, as private org members show up as CONTRIBUTOR).
//   - No untrusted value is interpolated into scripts: steps only pass data via env and outputs.
//   - GITHUB_TOKEN has read-only access to the repository contents. The push token is a GitHub App
//     installation token, created only when the validation succeeded, scoped to the current repository
//     and to contents and workflows permissions, and revoked at the end of the job.
//   - The GitHub App private key is an environment secret, available only to workflows running from the main branch.
//   - The exact validated PR head SHA is pushed, and the push fails if the PR head changed in between.
//   - For PRs from forks, an approval of the exact head commit by a user with write access is required,
//     as the checks of such PRs come from workflows the PR author controls.
func slashMergeWorkflow() *ghworkflow.Workflow {
	const (
		validateID = "validate"
		appTokenID = "app-token"
		pushID     = "push"

		validated = "steps." + validateID + ".outputs.sha != ''"
	)

	githubScript := func(name string) *ghworkflow.JobStep {
		return ghworkflow.Step(name).
			SetUsesWithComment(
				"actions/github-script@"+config.GitHubScriptActionRef,
				"version: "+config.GitHubScriptActionVersion,
			)
	}

	return &ghworkflow.Workflow{
		Name: slashMergeWorkflowName,
		IssueComment: &ghworkflow.IssueComment{
			Types: []string{"created"},
		},
		// deny everything by default, the job requests what it needs
		Permissions: ghworkflow.Permissions{},
		Jobs: map[string]*ghworkflow.Job{
			slashMergeWorkflowName: {
				// cheap pre-filter only, the exact command and the permissions are validated by the script
				If:     "github.event.issue.pull_request && startsWith(github.event.comment.body, '/nm')",
				RunsOn: ghworkflow.NewRunsOnGroupLabel(ghworkflow.GenericRunner, ""),
				Concurrency: &ghworkflow.Concurrency{
					Group:            slashMergeWorkflowName + "-${{ github.event.issue.number }}",
					CancelInProgress: false,
				},
				Environment: &ghworkflow.JobEnvironment{
					Name: slashMergeEnvironment,
				},
				TimeoutMinutes: 15,
				Permissions: ghworkflow.Permissions{
					"checks":        ghworkflow.PermissionActionRead,
					"contents":      ghworkflow.PermissionActionRead,
					"issues":        ghworkflow.PermissionActionWrite,
					"pull-requests": ghworkflow.PermissionActionWrite,
					"statuses":      ghworkflow.PermissionActionRead,
				},
				Steps: []*ghworkflow.JobStep{
					githubScript("validate").
						SetID(validateID).
						SetWith("script", slashMergeValidateScript),
					ghworkflow.Step("app token").
						SetID(appTokenID).
						SetCustomCondition(validated).
						SetUsesWithComment(
							"actions/create-github-app-token@"+config.CreateGitHubAppTokenActionRef,
							"version: "+config.CreateGitHubAppTokenActionVersion,
						).
						SetWith("client-id", "${{ vars."+slashMergeAppClientIDVariable+" }}").
						SetWith("private-key", "${{ secrets."+slashMergeAppPrivateKeySecret+" }}").
						// no owner/repositories: the token is scoped to the current repository
						SetWith("permission-contents", "write").
						SetWith("permission-workflows", "write"),
					ghworkflow.Step("checkout").
						SetCustomCondition(validated).
						SetUsesWithComment(
							"actions/checkout@"+config.CheckOutActionRef,
							"version: "+config.CheckOutActionVersion,
						).
						SetWith("ref", "${{ steps."+validateID+".outputs.base }}").
						SetWith("fetch-depth", "0").
						SetWith("filter", "blob:none").
						SetWith("token", "${{ steps."+appTokenID+".outputs.token }}"),
					ghworkflow.Step("fast-forward").
						SetID(pushID).
						SetCustomCondition(validated).
						SetEnv("BASE", "${{ steps."+validateID+".outputs.base }}").
						SetEnv("PR_SHA", "${{ steps."+validateID+".outputs.sha }}").
						SetEnv("PR_NUMBER", "${{ github.event.issue.number }}").
						SetCommand(slashMergeFastForwardScript),
					githubScript("report push failure").
						SetCustomCondition("failure() && steps."+pushID+".outcome == 'failure'").
						SetWith("script", slashMergeReportPushFailureScript),
					githubScript("react on success").
						SetCustomCondition("success() && "+validated).
						SetEnv("REACTION", "rocket").
						SetWith("script", slashMergeReactScript),
					githubScript("react on failure").
						SetCustomCondition("failure()").
						SetEnv("REACTION", "-1").
						SetWith("script", slashMergeReactScript),
				},
			},
		},
	}
}

// enableSlashMergeEnvironment makes sure the slash-merge environment exists, and only the main branch
// can deploy to it, so that the push token is not available to workflows running from other branches.
//
// The token itself is not managed by kres, it should be set as the environment secret manually.
//
//nolint:gocyclo,cyclop
func (r *Repository) enableSlashMergeEnvironment(client *github.Client) error {
	if r.DryRun {
		fmt.Printf("dry-run: repository would ensure environment %q exists, allowing deployments only from %q\n", slashMergeEnvironment, r.MainBranch)

		return nil
	}

	ctx := context.Background()
	owner, repo := r.meta.GitHubOrganization, r.meta.GitHubRepository

	env, resp, err := client.Repositories.GetEnvironment(ctx, owner, repo, slashMergeEnvironment)
	if err != nil && (resp == nil || resp.StatusCode != http.StatusNotFound) {
		return fmt.Errorf("failed to get environment %q: %w", slashMergeEnvironment, err)
	}

	if env == nil ||
		env.GetCanAdminsBypass() ||
		env.GetDeploymentBranchPolicy() == nil ||
		env.GetDeploymentBranchPolicy().GetProtectedBranches() ||
		!env.GetDeploymentBranchPolicy().GetCustomBranchPolicies() {
		if _, _, err = client.Repositories.CreateUpdateEnvironment(ctx, owner, repo, slashMergeEnvironment, &github.CreateUpdateEnvironment{
			CanAdminsBypass: new(false),
			DeploymentBranchPolicy: &github.BranchPolicy{
				ProtectedBranches:    new(false),
				CustomBranchPolicies: new(true),
			},
		}); err != nil {
			return fmt.Errorf("failed to create or update environment %q: %w", slashMergeEnvironment, err)
		}

		fmt.Printf("environment %q configured\n", slashMergeEnvironment)
	}

	var policies []*github.DeploymentBranchPolicy

	for policy, err := range client.Repositories.ListDeploymentBranchPoliciesIter(ctx, owner, repo, slashMergeEnvironment, &github.ListOptions{PerPage: 100}) {
		if err != nil {
			return fmt.Errorf("failed to list deployment branch policies of environment %q: %w", slashMergeEnvironment, err)
		}

		policies = append(policies, policy)
	}

	found := false

	for _, policy := range policies {
		if policy.GetName() == r.MainBranch && policy.GetType() == "branch" {
			found = true

			continue
		}

		// any other branch (or tag) could deploy to the environment, and get access to the push token
		if _, err = client.Repositories.DeleteDeploymentBranchPolicy(ctx, owner, repo, slashMergeEnvironment, policy.GetID()); err != nil {
			return fmt.Errorf("failed to delete deployment branch policy %q of environment %q: %w", policy.GetName(), slashMergeEnvironment, err)
		}

		fmt.Printf("deployment %s policy %q removed from environment %q\n", policy.GetType(), policy.GetName(), slashMergeEnvironment)
	}

	if !found {
		if _, _, err = client.Repositories.CreateDeploymentBranchPolicy(ctx, owner, repo, slashMergeEnvironment, github.CreateDeploymentBranchPolicyRequest{
			Name: r.MainBranch,
			Type: new("branch"),
		}); err != nil {
			return fmt.Errorf("failed to create deployment branch policy of environment %q: %w", slashMergeEnvironment, err)
		}

		fmt.Printf("environment %q restricted to branch %q\n", slashMergeEnvironment, r.MainBranch)
	}

	return nil
}

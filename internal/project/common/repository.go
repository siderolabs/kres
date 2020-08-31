// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"github.com/google/go-github/v32/github"

	"github.com/talos-systems/kres/internal/dag"
	"github.com/talos-systems/kres/internal/project/meta"
)

// Repository sets up repository settings.
type Repository struct {
	dag.BaseNode

	meta *meta.Options

	MainBranch     string   `yaml:"mainBranch"`
	EnforeContexts []string `yaml:"enforceContexts"`
}

// NewRepository initializes Repository.
func NewRepository(meta *meta.Options) *Repository {
	return &Repository{
		BaseNode: dag.NewBaseNode("repository"),

		meta: meta,

		MainBranch: "master",
		EnforeContexts: []string{
			"continuous-integration/drone/pr",
			"status-lgtm",
		},
	}
}

// CompileGitHub implements github.Compiler.
func (r *Repository) CompileGitHub(client *github.Client) error {
	branchProtection, resp, err := client.Repositories.GetBranchProtection(context.Background(), r.meta.GitHubOrganization, r.meta.GitHubRepository, r.MainBranch)
	if err != nil {
		if resp.StatusCode != http.StatusNotFound {
			return err
		}
	}

	req := github.ProtectionRequest{
		RequireLinearHistory: github.Bool(true),
		AllowDeletions:       github.Bool(false),
		AllowForcePushes:     github.Bool(false),
		EnforceAdmins:        false,

		RequiredPullRequestReviews: &github.PullRequestReviewsEnforcementRequest{
			RequiredApprovingReviewCount: 1,
			DismissStaleReviews:          true,
		},

		RequiredStatusChecks: &github.RequiredStatusChecks{
			Strict:   true,
			Contexts: r.EnforeContexts,
		},
	}

	if branchProtection != nil {
		// check if everything is already set up
		if branchProtection.GetAllowDeletions().Enabled == *req.AllowDeletions &&
			branchProtection.GetAllowForcePushes().Enabled == *req.AllowForcePushes &&
			branchProtection.GetEnforceAdmins().Enabled == req.EnforceAdmins &&
			branchProtection.GetRequireLinearHistory().Enabled == *req.RequireLinearHistory &&
			branchProtection.GetRequiredPullRequestReviews() != nil &&
			branchProtection.GetRequiredPullRequestReviews().DismissStaleReviews == req.RequiredPullRequestReviews.DismissStaleReviews &&
			branchProtection.GetRequiredPullRequestReviews().RequiredApprovingReviewCount == req.RequiredPullRequestReviews.RequiredApprovingReviewCount &&
			branchProtection.GetRequiredStatusChecks() != nil &&
			branchProtection.GetRequiredStatusChecks().Strict == req.RequiredStatusChecks.Strict &&
			equalStringSlices(branchProtection.GetRequiredStatusChecks().Contexts, req.RequiredStatusChecks.Contexts) {
			return nil
		}
	}

	_, _, err = client.Repositories.UpdateBranchProtection(context.Background(), r.meta.GitHubOrganization, r.meta.GitHubRepository, r.MainBranch, &req)
	if err != nil {
		return err
	}

	fmt.Println("branch protection settings updated")

	return nil
}

func equalStringSlices(a, b []string) bool {
	a = append([]string(nil), a...)
	b = append([]string(nil), b...)

	sort.Strings(a)
	sort.Strings(b)

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

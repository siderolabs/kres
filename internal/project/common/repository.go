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
	"github.com/talos-systems/kres/internal/output/conform"
	"github.com/talos-systems/kres/internal/project/meta"
)

// Repository sets up repository settings.
type Repository struct {
	dag.BaseNode

	meta *meta.Options

	MainBranch      string   `yaml:"mainBranch"`
	EnforceContexts []string `yaml:"enforceContexts"`

	EnableConform     bool     `yaml:"enableConform"`
	ConformWebhookURL string   `yaml:"conformWebhookURL"`
	ConformTypes      []string `yaml:"conformTypes"`
	ConformScopes     []string `yaml:"conformScopes"`

	BotName string `yaml:"botName"`
}

// NewRepository initializes Repository.
func NewRepository(meta *meta.Options) *Repository {
	return &Repository{
		BaseNode: dag.NewBaseNode("repository"),

		meta: meta,

		MainBranch: "master",
		EnforceContexts: []string{
			"continuous-integration/drone/pr",
		},

		EnableConform:     true,
		ConformWebhookURL: "https://conform.dev.talos-systems.io/github",
		ConformTypes: []string{
			"chore",
			"docs",
			"perf",
			"refactor",
			"style",
			"test",
			"release",
		},
		ConformScopes: []string{
			"*",
		},

		BotName: "talos-bot",
	}
}

// CompileConform implements conform.Compiler.
func (r *Repository) CompileConform(o *conform.Output) error {
	if !r.EnableConform {
		return nil
	}

	o.Enable()
	o.SetScopes(r.ConformScopes)
	o.SetTypes(r.ConformTypes)

	return nil
}

// CompileGitHub implements github.Compiler.
func (r *Repository) CompileGitHub(client *github.Client) error {
	if err := r.enableBranchProtection(client); err != nil {
		return err
	}

	if r.EnableConform {
		if err := r.enableConform(client); err != nil {
			return err
		}
	}

	if err := r.inviteBot(client); err != nil {
		return err
	}

	return nil
}

func (r *Repository) enableBranchProtection(client *github.Client) error {
	branchProtection, resp, err := client.Repositories.GetBranchProtection(context.Background(), r.meta.GitHubOrganization, r.meta.GitHubRepository, r.MainBranch)
	if err != nil {
		if resp.StatusCode != http.StatusNotFound {
			return err
		}
	}

	enforceContexts := r.EnforceContexts
	if r.EnableConform {
		enforceContexts = append(enforceContexts,
			"conform/commit/commit-body",
			"conform/commit/conventional-commit",
			"conform/commit/dco",
			"conform/commit/header-case",
			"conform/commit/header-last-character",
			"conform/commit/header-length",
			"conform/commit/imperative-mood",
			"conform/commit/number-of-commits",
			"conform/commit/spellcheck",
			"conform/license/file-header",
		)
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
			Contexts: enforceContexts,
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

func (r *Repository) enableConform(client *github.Client) error {
	hooks, _, err := client.Repositories.ListHooks(context.Background(), r.meta.GitHubOrganization, r.meta.GitHubRepository, &github.ListOptions{})
	if err != nil {
		return err
	}

	for _, hook := range hooks {
		if hook.Config["url"].(string) == r.ConformWebhookURL {
			return nil
		}
	}

	_, _, err = client.Repositories.CreateHook(context.Background(), r.meta.GitHubOrganization, r.meta.GitHubRepository, &github.Hook{
		Active: github.Bool(true),
		Config: map[string]interface{}{
			"url":          r.ConformWebhookURL,
			"content_type": "json",
			"insecure_ssl": "0",
		},
		Events: []string{
			"push",
			"pull_request",
		},
	})

	if err != nil {
		return err
	}

	fmt.Println("conform webhook created")

	return nil
}

func (r *Repository) inviteBot(client *github.Client) error {
	users, _, err := client.Repositories.ListCollaborators(context.Background(), r.meta.GitHubOrganization, r.meta.GitHubRepository, &github.ListCollaboratorsOptions{})
	if err != nil {
		return err
	}

	for _, user := range users {
		if user.GetLogin() == r.BotName {
			return nil
		}
	}

	_, resp, err := client.Repositories.AddCollaborator(context.Background(), r.meta.GitHubOrganization, r.meta.GitHubRepository, r.BotName, &github.RepositoryAddCollaboratorOptions{
		Permission: "maintain",
	})
	if err != nil {
		if resp.StatusCode == http.StatusNoContent {
			return nil
		}

		return err
	}

	fmt.Println("invited bot", r.BotName)

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

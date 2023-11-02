// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"github.com/google/go-github/v55/github"
	"github.com/siderolabs/gen/xslices"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/conform"
	"github.com/siderolabs/kres/internal/output/license"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Repository sets up repository settings.
type Repository struct { //nolint:govet,maligned
	dag.BaseNode

	meta *meta.Options

	MainBranch      string   `yaml:"mainBranch"`
	EnforceContexts []string `yaml:"enforceContexts"`

	EnableConform            bool     `yaml:"enableConform"`
	ConformWebhookURL        string   `yaml:"conformWebhookURL"`
	ConformTypes             []string `yaml:"conformTypes"`
	ConformScopes            []string `yaml:"conformScopes"`
	ConformLicenseCheck      bool     `yaml:"conformLicenseCheck"`
	ConformGPGSignatureCheck bool     `yaml:"conformGPGSignatureCheck"`

	DeprecatedEnableLicense *bool `yaml:"enableLicense"`

	License LicenseConfig `yaml:"license"`

	BotName string `yaml:"botName"`
}

// LicenseConfig configures the license.
//
//nolint:govet
type LicenseConfig struct {
	Enabled bool           `yaml:"enabled"`
	ID      string         `yaml:"id"`
	Params  map[string]any `yaml:"params"`
	Header  string         `yaml:"header"`
}

// NewRepository initializes Repository.
func NewRepository(meta *meta.Options) *Repository {
	return &Repository{
		BaseNode: dag.NewBaseNode("repository"),

		meta: meta,

		MainBranch: meta.MainBranch,

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
			".*",
		},
		ConformLicenseCheck:      true,
		ConformGPGSignatureCheck: true,

		License: LicenseConfig{
			Enabled: true,
			ID:      "MPL-2.0",
			Header:  mplHeader,
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
	o.SetLicenseCheck(r.ConformLicenseCheck)
	o.SetLicenseHeader(r.License.Header)
	o.SetGPGSignatureCheck(r.ConformGPGSignatureCheck)
	o.SetGitHubOrganization(r.meta.GitHubOrganization)

	return nil
}

// CompileLicense implements license.Compiler.
func (r *Repository) CompileLicense(o *license.Output) error {
	if r.meta.ContainerImageFrontend != config.ContainerImageFrontendDockerfile {
		return nil
	}

	if r.DeprecatedEnableLicense != nil {
		r.License.Enabled = *r.DeprecatedEnableLicense
	}

	if !r.License.Enabled {
		return nil
	}

	o.SetLicenseHeader(r.License.Header)

	return o.Enable(r.License.ID, r.License.Params)
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

	return r.inviteBot(client)
}

//nolint:gocyclo,cyclop
func (r *Repository) enableBranchProtection(client *github.Client) error {
	branchProtection, resp, err := client.Repositories.GetBranchProtection(context.Background(), r.meta.GitHubOrganization, r.meta.GitHubRepository, r.MainBranch)
	if err != nil {
		if resp == nil || resp.StatusCode != http.StatusNotFound {
			return err
		}
	}

	enforceContexts := r.EnforceContexts

	switch r.meta.CIProvider {
	case config.CIProviderDrone:
		enforceContexts = append(enforceContexts, "continuous-integration/drone/pr")
	case config.CIProviderGitHubActions:
		enforceContexts = append(enforceContexts, "default")
	}

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
		)

		if r.ConformLicenseCheck {
			enforceContexts = append(enforceContexts,
				"conform/license/file-header",
			)
		}
	}

	req := github.ProtectionRequest{
		RequireLinearHistory: github.Bool(true),
		AllowDeletions:       github.Bool(false),
		AllowForcePushes:     github.Bool(false),
		EnforceAdmins:        true,

		RequiredPullRequestReviews: &github.PullRequestReviewsEnforcementRequest{
			RequiredApprovingReviewCount: 1,
		},

		RequiredStatusChecks: &github.RequiredStatusChecks{
			Strict: true,
			Checks: xslices.Map(enforceContexts, func(c string) *github.RequiredStatusCheck {
				return &github.RequiredStatusCheck{
					Context: c,
				}
			}),
		},
	}

	if branchProtection != nil {
		var sigProtected *github.SignaturesProtectedBranch

		sigProtected, _, err = client.Repositories.GetSignaturesProtectedBranch(context.Background(), r.meta.GitHubOrganization, r.meta.GitHubRepository, r.MainBranch)
		if err != nil {
			return nil //nolint:nilerr
		}

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
			equalStringSlices(
				xslices.Map(branchProtection.GetRequiredStatusChecks().Checks,
					func(s *github.RequiredStatusCheck) string {
						return s.Context
					}), enforceContexts) &&
			sigProtected.GetEnabled() {
			return nil
		}
	}

	_, _, err = client.Repositories.UpdateBranchProtection(context.Background(), r.meta.GitHubOrganization, r.meta.GitHubRepository, r.MainBranch, &req)
	if err != nil {
		return err
	}

	_, _, err = client.Repositories.RequireSignaturesOnProtectedBranch(context.Background(), r.meta.GitHubOrganization, r.meta.GitHubRepository, r.MainBranch)
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
		if hook.Config["url"].(string) == r.ConformWebhookURL { //nolint:forcetypeassert
			return nil
		}
	}

	_, _, err = client.Repositories.CreateHook(context.Background(), r.meta.GitHubOrganization, r.meta.GitHubRepository, &github.Hook{
		Active: github.Bool(true),
		Config: map[string]any{
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

const mplHeader = `// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
`

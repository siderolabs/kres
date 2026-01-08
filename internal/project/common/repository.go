// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/google/go-github/v81/github"
	"github.com/siderolabs/gen/xslices"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output"
	"github.com/siderolabs/kres/internal/output/conform"
	"github.com/siderolabs/kres/internal/output/conform/licensepolicy"
	"github.com/siderolabs/kres/internal/output/license"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Repository sets up repository settings.
type Repository struct { //nolint:govet
	dag.BaseNode

	meta *meta.Options

	MainBranch      string   `yaml:"mainBranch"`
	EnforceContexts []string `yaml:"enforceContexts"`

	EnableConform             bool     `yaml:"enableConform"`
	ConformWebhookURL         string   `yaml:"conformWebhookURL"`
	ConformTypes              []string `yaml:"conformTypes"`
	ConformScopes             []string `yaml:"conformScopes"`
	ConformLicenseCheck       bool     `yaml:"conformLicenseCheck"`
	ConformGPGSignatureCheck  bool     `yaml:"conformGPGSignatureCheck"`
	ConformMaximumOfOneCommit bool     `yaml:"conformMaximumOfOneCommit"`

	DeprecatedEnableLicense *bool `yaml:"enableLicense"`

	DeprecatedLicense *LicenseConfig `yaml:"license"`

	Licenses      []LicenseConfig      `yaml:"licenses"`
	LicenseChecks []licensepolicy.Spec `yaml:"licenseChecks"`

	BotName string `yaml:"botName"`

	SkipStaleWorkflow bool `yaml:"skipStaleWorkflow"`
}

// LicenseConfig configures the license.
type LicenseConfig struct {
	Enabled *bool          `yaml:"enabled"`
	Root    string         `yaml:"root"`
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
		ConformLicenseCheck:       true,
		ConformGPGSignatureCheck:  true,
		ConformMaximumOfOneCommit: false,

		Licenses: []LicenseConfig{
			{
				ID:     "MPL-2.0",
				Header: output.License(MPLHeader, "// "),
				Root:   ".",
			},
		},

		BotName: "talos-bot",
	}
}

// AfterLoad maps back main branch override to meta.
func (r *Repository) AfterLoad() error {
	r.meta.MainBranch = r.MainBranch
	r.meta.SkipStaleWorkflow = r.SkipStaleWorkflow

	return nil
}

// CompileConform implements conform.Compiler.
func (r *Repository) CompileConform(o *conform.Output) error {
	if !r.EnableConform {
		return nil
	}

	// If license specs are not defined, generate the default one from the licenses section
	if r.LicenseChecks == nil {
		r.LicenseChecks = xslices.Map(r.Licenses, func(lc LicenseConfig) licensepolicy.Spec {
			return licensepolicy.Spec{
				Root:   lc.Root,
				Header: lc.Header,
			}
		})
	}

	o.Enable()
	o.SetScopes(r.ConformScopes)
	o.SetTypes(r.ConformTypes)
	o.SetLicenseCheck(r.ConformLicenseCheck)
	o.SetGPGSignatureCheck(r.ConformGPGSignatureCheck)
	o.SetGitHubOrganization(r.meta.GitHubOrganization)
	o.SetLicensePolicySpecs(r.LicenseChecks)
	o.SetMaximumOfOneCommit(r.ConformMaximumOfOneCommit)

	return nil
}

// CompileLicense implements license.Compiler.
func (r *Repository) CompileLicense(o *license.Output) error {
	if r.meta.ContainerImageFrontend != config.ContainerImageFrontendDockerfile {
		return nil
	}

	if r.DeprecatedLicense != nil {
		// prepend to licenses
		r.Licenses = append([]LicenseConfig{*r.DeprecatedLicense}, r.Licenses...)
	}

	for _, lcs := range r.Licenses {
		if r.DeprecatedEnableLicense != nil {
			lcs.Enabled = r.DeprecatedEnableLicense
		}

		// If enabled is unset (nil), default to true.
		if lcs.Enabled != nil && !*lcs.Enabled {
			continue
		}

		if err := o.Enable(lcs.Root, lcs.ID, lcs.Params); err != nil {
			return err
		}

		o.SetLicenseHeader(lcs.Root, lcs.Header)
	}

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
		if !r.meta.CompileGithubWorkflowsOnly {
			enforceContexts = append(enforceContexts, "default")
		}
	}

	enforceContexts = append(enforceContexts, r.meta.ExtraEnforcedContexts...)

	if r.EnableConform {
		enforceContexts = append(enforceContexts,
			"conform/commit/commit-body",
			"conform/commit/conventional-commit",
			"conform/commit/dco",
			"conform/commit/header-case",
			"conform/commit/header-last-character",
			"conform/commit/header-length",
			"conform/commit/imperative-mood",
			"conform/commit/spellcheck",
		)

		if r.ConformMaximumOfOneCommit {
			enforceContexts = append(enforceContexts, "conform/commit/number-of-commits")
		}

		if r.ConformLicenseCheck {
			enforceContexts = append(enforceContexts,
				"conform/license/file-header",
			)
		}
	}

	statusChecks := xslices.Map(enforceContexts, func(c string) *github.RequiredStatusCheck {
		return &github.RequiredStatusCheck{
			Context: c,
		}
	})

	req := github.ProtectionRequest{
		RequireLinearHistory: github.Ptr(true),
		AllowDeletions:       github.Ptr(false),
		AllowForcePushes:     github.Ptr(false),
		EnforceAdmins:        true,

		RequiredPullRequestReviews: &github.PullRequestReviewsEnforcementRequest{
			RequiredApprovingReviewCount: 1,
		},

		RequiredStatusChecks: &github.RequiredStatusChecks{
			Strict: true,
			Checks: &statusChecks,
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
				xslices.Map(*branchProtection.GetRequiredStatusChecks().Checks,
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
		if *hook.Config.URL == r.ConformWebhookURL {
			return nil
		}
	}

	_, _, err = client.Repositories.CreateHook(context.Background(), r.meta.GitHubOrganization, r.meta.GitHubRepository, &github.Hook{
		Active: github.Ptr(true),
		Config: &github.HookConfig{
			URL:         &r.ConformWebhookURL,
			ContentType: github.Ptr("json"),
			InsecureSSL: github.Ptr("0"),
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
	a = slices.Clone(a)
	b = slices.Clone(b)

	slices.Sort(a)
	slices.Sort(b)

	return slices.Equal(a, b)
}

// MPLHeader is the Mozilla Public License 2.0 header.
const MPLHeader = `This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at http://mozilla.org/MPL/2.0/.`

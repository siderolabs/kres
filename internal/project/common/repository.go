// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/google/go-github/v84/github"
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

	// autoContextsFunc returns additional status check contexts to enforce,
	// computed at compile time (e.g. from GHWorkflow jobs).
	autoContextsFunc func() []string

	// autoLabelsFunc returns PR trigger labels that should exist on the
	// repository, mapped to their description (empty string if unset). Computed
	// at compile time (e.g. from GHWorkflow jobs).
	autoLabelsFunc func() map[string]string

	MainBranch string `yaml:"mainBranch"`

	// DryRun, when true, logs the intended branch protection / conform changes
	// instead of calling the GitHub API. Dry-run is also implicit when
	// GITHUB_TOKEN is unset.
	DryRun bool `yaml:"dryRun,omitempty"`

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

// CompileGitHub implements github.Compiler. When Repository.DryRun is true,
// changes are logged instead of applied. When GITHUB_TOKEN is unset and
// DryRun is false, the step is skipped entirely.
func (r *Repository) CompileGitHub(client *github.Client) error {
	if client == nil && !r.DryRun {
		return nil
	}

	if err := r.enableBranchProtection(client); err != nil {
		return err
	}

	if err := r.enableLabels(client); err != nil {
		return err
	}

	if r.DryRun {
		return nil
	}

	if r.EnableConform {
		if err := r.enableConform(client); err != nil {
			return err
		}
	}

	return r.inviteBot(client)
}

func (r *Repository) enableBranchProtection(client *github.Client) error {
	// When kres runs from a release-* branch, protect that branch; otherwise
	// protect main.
	targetBranch := r.MainBranch
	if strings.HasPrefix(r.meta.CurrentBranch, "release-") {
		targetBranch = r.meta.CurrentBranch
	}

	enforceContexts := r.buildEnforceContexts(nil)

	if r.DryRun {
		fmt.Printf("dry-run: branch protection for %s would require %d contexts:\n", targetBranch, len(enforceContexts))

		for _, c := range enforceContexts {
			fmt.Printf("  - %s\n", c)
		}

		return nil
	}

	return r.applyBranchProtection(client, targetBranch, enforceContexts)
}

// SetAutoContextsFunc registers a callback that supplies auto-computed status
// check contexts (e.g. from GHWorkflow jobs). The callback is invoked at compile
// time, after the DAG has been loaded from .kres.yaml.
func (r *Repository) SetAutoContextsFunc(fn func() []string) {
	r.autoContextsFunc = fn
}

// SetAutoLabelsFunc registers a callback that supplies auto-computed PR labels
// (e.g. trigger labels from GHWorkflow jobs) mapped to their description. The
// callback is invoked at compile time, after the DAG has been loaded from
// .kres.yaml.
func (r *Repository) SetAutoLabelsFunc(fn func() map[string]string) {
	r.autoLabelsFunc = fn
}

// autoLabelColor is the default color for labels created automatically by kres.
const autoLabelColor = "ededed"

func (r *Repository) enableLabels(client *github.Client) error {
	if r.autoLabelsFunc == nil {
		return nil
	}

	labels := r.autoLabelsFunc()
	if len(labels) == 0 {
		return nil
	}

	names := slices.Sorted(maps.Keys(labels))

	if r.DryRun {
		fmt.Printf("dry-run: repository would ensure %d PR labels exist:\n", len(names))

		for _, name := range names {
			if desc := labels[name]; desc != "" {
				fmt.Printf("  - %s — %s\n", name, desc)
			} else {
				fmt.Printf("  - %s\n", name)
			}
		}

		return nil
	}

	for _, name := range names {
		desc := labels[name]

		existing, resp, err := client.Issues.GetLabel(context.Background(), r.meta.GitHubOrganization, r.meta.GitHubRepository, name)
		if err != nil {
			if resp == nil || resp.StatusCode != http.StatusNotFound {
				return fmt.Errorf("failed to check label %q: %w", name, err)
			}

			if _, _, err := client.Issues.CreateLabel(context.Background(), r.meta.GitHubOrganization, r.meta.GitHubRepository, &github.Label{
				Name:        new(name),
				Color:       new(autoLabelColor),
				Description: new(desc),
			}); err != nil {
				return fmt.Errorf("failed to create label %q: %w", name, err)
			}

			continue
		}

		if desc != "" && existing.GetDescription() != desc {
			if _, _, err := client.Issues.EditLabel(context.Background(), r.meta.GitHubOrganization, r.meta.GitHubRepository, name, &github.Label{
				Description: new(desc),
			}); err != nil {
				return fmt.Errorf("failed to update description for label %q: %w", name, err)
			}
		}
	}

	return nil
}

func (r *Repository) buildEnforceContexts(base []string) []string {
	enforceContexts := base

	if r.autoContextsFunc != nil {
		enforceContexts = append(enforceContexts, r.autoContextsFunc()...)
	}

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

	return slices.Compact(slices.Sorted(slices.Values(enforceContexts)))
}

//nolint:gocyclo,cyclop
func (r *Repository) applyBranchProtection(client *github.Client, branch string, enforceContexts []string) error {
	branchProtection, resp, err := client.Repositories.GetBranchProtection(context.Background(), r.meta.GitHubOrganization, r.meta.GitHubRepository, branch)
	if err != nil {
		if resp == nil || resp.StatusCode != http.StatusNotFound {
			return err
		}
	}

	statusChecks := xslices.Map(enforceContexts, func(c string) *github.RequiredStatusCheck {
		return &github.RequiredStatusCheck{
			Context: c,
		}
	})

	req := github.ProtectionRequest{
		RequireLinearHistory: new(true),
		AllowDeletions:       new(false),
		AllowForcePushes:     new(false),
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
		sigProtected, _, sigErr := client.Repositories.GetSignaturesProtectedBranch(context.Background(), r.meta.GitHubOrganization, r.meta.GitHubRepository, branch)
		if sigErr != nil {
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

	_, updateResp, err := client.Repositories.UpdateBranchProtection(context.Background(), r.meta.GitHubOrganization, r.meta.GitHubRepository, branch, &req)
	if err != nil {
		if updateResp != nil && updateResp.StatusCode == http.StatusNotFound {
			fmt.Printf("branch %s not found, skipping protection\n", branch)

			return nil
		}

		return err
	}

	if _, _, err = client.Repositories.RequireSignaturesOnProtectedBranch(context.Background(), r.meta.GitHubOrganization, r.meta.GitHubRepository, branch); err != nil {
		return err
	}

	fmt.Printf("branch protection settings updated for %s\n", branch)

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
		Active: new(true),
		Config: &github.HookConfig{
			URL:         &r.ConformWebhookURL,
			ContentType: new("json"),
			InsecureSSL: new("0"),
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

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"context"
	_ "embed"
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/google/go-github/v92/github"
	"github.com/siderolabs/gen/xslices"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output"
	"github.com/siderolabs/kres/internal/output/conform"
	"github.com/siderolabs/kres/internal/output/conform/licensepolicy"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/output/lefthook"
	"github.com/siderolabs/kres/internal/output/license"
	"github.com/siderolabs/kres/internal/output/template"
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
	// instead of calling the GitHub API. Must be set explicitly; when
	// GITHUB_TOKEN is unset, GitHub API integration is skipped entirely.
	DryRun bool `yaml:"dryRun,omitempty"`

	// AllowPushesToMain relaxes the branch protection rules so that commits can
	// be pushed directly to the protected branch, bypassing pull requests: the
	// required pull request reviews and required status checks are dropped,
	// while the other protections (deletion, force pushes, linear history,
	// signatures) stay in place.
	AllowPushesToMain bool `yaml:"allowPushesToMain,omitempty"`

	EnableImmutableReleases bool `yaml:"enableImmutableReleases"`
	EnableConform           bool `yaml:"enableConform"`
	// ConformWebhookURL is the URL of the legacy conform webhook. Conform now
	// runs as a GitHub Actions workflow, and a webhook matching this URL is
	// removed from the repository.
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

	EnableLefthook bool `yaml:"enableLefthook"`

	EnableAIPolicy bool `yaml:"enableAIPolicy"`

	ProjectAutoAdd ProjectAutoAdd `yaml:"projectAutoAdd"`
}

// ProjectAutoAdd configures the workflow which adds pull requests to a GitHub organization project.
//
// The workflow authenticates as a GitHub App installed in the organization with
// the "Projects: Read & write" organization permission and the "Pull requests: Read"
// repository permission.
type ProjectAutoAdd struct {
	// Organization owning the project, defaults to the repository organization.
	Organization string `yaml:"organization"`
	// StatusField is the name of the single select field to set the status of the item.
	StatusField string `yaml:"statusField"`
	// DraftStatus is the status of draft pull requests.
	DraftStatus string `yaml:"draftStatus"`
	// ReadyStatus is the status of pull requests ready for review.
	ReadyStatus string `yaml:"readyStatus"`
	// IterationField is the name of the iteration field; opened pull requests are assigned the current iteration.
	IterationField string `yaml:"iterationField"`
	// AppClientIDSecret is the name of the secret with the GitHub App client ID.
	AppClientIDSecret string `yaml:"appClientIDSecret"`
	// AppPrivateKeySecret is the name of the secret with the GitHub App private key.
	AppPrivateKeySecret string `yaml:"appPrivateKeySecret"`
	// ProjectNumber is the number of the project in the organization.
	ProjectNumber int `yaml:"projectNumber"`
	// Enabled enables the workflow (enabled by default).
	Enabled bool `yaml:"enabled"`
}

// aiPolicy is the "Human Responsible" policy from https://github.com/43081j/ai-policy.
//
//go:embed resources/AI_POLICY.md
var aiPolicy string

// projectAutoAddScript is the github-script which adds the pull request to the project.
//
//go:embed resources/project-auto-add.js
var projectAutoAddScript string

// secretNameRe matches valid GitHub Actions secret names (the reserved GITHUB_ prefix is checked separately).
var secretNameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// organizationNameRe matches valid GitHub organization names.
var organizationNameRe = regexp.MustCompile(`^[A-Za-z0-9](?:-?[A-Za-z0-9]){0,38}$`)

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

		EnableImmutableReleases: true,

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
				Header: output.License(license.MPLHeader, "// "),
				Root:   ".",
			},
		},

		BotName: "talos-bot",

		EnableLefthook: true,

		EnableAIPolicy: true,

		ProjectAutoAdd: ProjectAutoAdd{
			Enabled:             true,
			ProjectNumber:       17,
			StatusField:         "Status",
			DraftStatus:         "In Progress",
			ReadyStatus:         "In Review",
			IterationField:      "Iteration",
			AppClientIDSecret:   "PROJECT_APP_CLIENT_ID",
			AppPrivateKeySecret: "PROJECT_APP_PRIVATE_KEY",
		},
	}
}

// AfterLoad maps back main branch override to meta.
func (r *Repository) AfterLoad() error {
	r.meta.MainBranch = r.MainBranch
	r.meta.SkipStaleWorkflow = r.SkipStaleWorkflow

	return nil
}

// CompileLefthook implements lefthook.Compiler.
func (r *Repository) CompileLefthook(o *lefthook.Output) error {
	if !r.EnableLefthook {
		return nil
	}

	o.Enable()

	return nil
}

// CompileTemplates implements template.Compiler.
func (r *Repository) CompileTemplates(o *template.Output) error {
	if !r.EnableAIPolicy {
		return nil
	}

	o.Define("AI_POLICY.md", aiPolicy).NoPreamble()

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

// conformJobName is the name of the conform workflow and its job, which is also
// the status check context required by branch protection.
const conformJobName = "conform"

// conformInstallScript downloads the conform release binary for the runner architecture
// into $RUNNER_TEMP and verifies it against the release checksums.
const conformInstallScript = `case "$(uname -m)" in
  x86_64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

cd "${RUNNER_TEMP}"

url="https://github.com/siderolabs/conform/releases/download/${CONFORM_VERSION}"
curl -fsSL --retry 3 -o "conform-linux-${arch}" "${url}/conform-linux-${arch}"
curl -fsSL --retry 3 -o conform-sha256sum.txt "${url}/sha256sum.txt"
grep " conform-linux-${arch}$" conform-sha256sum.txt | sha256sum -c -

install -m 0755 "conform-linux-${arch}" conform`

// skipBotPullRequests is the job condition which skips pull requests opened by the dependency update bots.
//
// The condition is based on the author of the pull request (not on github.actor), so that the result
// is the same for every event of the pull request, no matter who triggered it.
const skipBotPullRequests = "github.event.pull_request.user.login != 'dependabot[bot]' && github.event.pull_request.user.login != 'renovate[bot]'"

// CompileGitHubWorkflow implements ghworkflow.Compiler.
//
// It generates the conform workflow which enforces the policies from .conform.yaml
// on every commit of a pull request. The workflow needs only read access, so it works
// for pull requests from forks as well. The job fails if any policy check fails, so
// the job itself is the single status check required by branch protection.
func (r *Repository) CompileGitHubWorkflow(o *ghworkflow.Output) error {
	if err := r.compileProjectAutoAddWorkflow(o); err != nil {
		return err
	}

	if !r.EnableConform {
		return nil
	}

	o.AddWorkflow(
		conformJobName,
		&ghworkflow.Workflow{
			Name:             conformJobName,
			Group:            "${{ github.workflow }}-${{ github.event.pull_request.number || github.run_id }}",
			CancelInProgress: true,
			PullRequest: ghworkflow.PullRequest{
				Branches: ghworkflow.Branches{
					r.meta.MainBranch,
					"release-*",
				},
			},
			Permissions: ghworkflow.Permissions{
				"contents": ghworkflow.PermissionActionRead,
			},
			Jobs: map[string]*ghworkflow.Job{
				conformJobName: {
					RunsOn: ghworkflow.NewRunsOnGroupLabel(ghworkflow.GenericRunner, ""),
					// commits of the bots are not expected to follow the policies
					If: skipBotPullRequests,
					Steps: []*ghworkflow.JobStep{
						// Check out the PR head commit (not the synthetic merge commit) with full history,
						// so that every commit of the PR can be checked against the base branch.
						ghworkflow.Step("checkout").
							SetUsesWithComment(
								"actions/checkout@"+config.CheckOutActionRef,
								"version: "+config.CheckOutActionVersion,
							).
							SetWith("ref", "${{ github.event.pull_request.head.sha }}").
							SetWith("fetch-depth", "0").
							SetWith("persist-credentials", "false"),
						// conform releases are immutable, so the checksum file from the release pins the binary.
						ghworkflow.Step("install conform").
							SetEnv("CONFORM_VERSION", config.ConformVersion).
							SetCommand(conformInstallScript),
						ghworkflow.Step("conform").
							SetEnv("BASE_REF", "${{ github.base_ref }}").
							// the token is used for GitHub API calls in the GPG identity check (higher rate limits),
							// it is read-only for pull requests from forks, which is sufficient
							SetEnv("GITHUB_TOKEN", "${{ secrets.GITHUB_TOKEN }}").
							SetCommand(`"${RUNNER_TEMP}/conform" enforce --base-branch="origin/${BASE_REF}" --commit-ref="refs/remotes/origin/${BASE_REF}"`),
					},
				},
			},
		},
	)

	return nil
}

// projectAutoAddWorkflowName is the name of the project auto-add workflow and its job.
const projectAutoAddWorkflowName = "project-auto-add"

// compileProjectAutoAddWorkflow generates the workflow which adds pull requests to the
// organization project, puts them into the current iteration and sets the status based
// on the draft state of the pull request.
//
// The workflow runs on pull_request_target, so that the GitHub App secrets are available
// for pull requests from forks as well: this is safe, as the workflow never checks out
// or runs the code from the pull request.
func (r *Repository) compileProjectAutoAddWorkflow(o *ghworkflow.Output) error {
	cfg := r.ProjectAutoAdd

	if !cfg.Enabled {
		return nil
	}

	organization := cfg.Organization
	if organization == "" {
		organization = r.meta.GitHubOrganization
	}

	// not a GitHub repository
	if organization == "" {
		return nil
	}

	if cfg.ProjectNumber <= 0 {
		return fmt.Errorf("projectAutoAdd: invalid project number %d", cfg.ProjectNumber)
	}

	// the values below are written into the workflow as is, so they should never be evaluated as expressions:
	// the workflow runs on pull_request_target with the secrets available
	if !organizationNameRe.MatchString(organization) {
		return fmt.Errorf("projectAutoAdd: invalid organization %q", organization)
	}

	for _, field := range []struct{ name, value string }{
		{"statusField", cfg.StatusField},
		{"draftStatus", cfg.DraftStatus},
		{"readyStatus", cfg.ReadyStatus},
		{"iterationField", cfg.IterationField},
	} {
		if field.value == "" {
			return fmt.Errorf("projectAutoAdd: %s is not set", field.name)
		}

		if strings.Contains(field.value, "${{") {
			return fmt.Errorf("projectAutoAdd: %s %q should not contain expressions", field.name, field.value)
		}
	}

	for _, secret := range []struct{ name, value string }{
		{"appClientIDSecret", cfg.AppClientIDSecret},
		{"appPrivateKeySecret", cfg.AppPrivateKeySecret},
	} {
		// GitHub reserves the GITHUB_ prefix, such secrets can't be created
		if !secretNameRe.MatchString(secret.value) || strings.HasPrefix(strings.ToUpper(secret.value), "GITHUB_") {
			return fmt.Errorf("projectAutoAdd: invalid %s %q", secret.name, secret.value)
		}
	}

	tokenStep := ghworkflow.Step("generate token").
		SetID("app-token").
		SetUsesWithComment(
			"actions/create-github-app-token@"+config.CreateGitHubAppTokenActionRef,
			"version: "+config.CreateGitHubAppTokenActionVersion,
		).
		SetWith("client-id", "${{ secrets."+cfg.AppClientIDSecret+" }}").
		SetWith("private-key", "${{ secrets."+cfg.AppPrivateKeySecret+" }}").
		SetWith("owner", organization).
		SetWith("permission-organization-projects", "write").
		SetWith("permission-pull-requests", "read")

	// Scope the token to the repository of the pull request instead of every repository the App is installed on.
	//
	// If the project is in a different organization, the token comes from the installation in that organization,
	// which has no access to this repository: this works only for public repositories.
	if strings.EqualFold(organization, r.meta.GitHubOrganization) {
		tokenStep.SetWith("repositories", "${{ github.event.repository.name }}")
	}

	o.AddWorkflow(
		projectAutoAddWorkflowName,
		&ghworkflow.Workflow{
			Name: projectAutoAddWorkflowName,
			// every run is idempotent and reads the draft state from its own event, so the latest run wins
			Group:            "${{ github.workflow }}-${{ github.event.pull_request.number }}",
			CancelInProgress: true,
			PullRequestTarget: ghworkflow.PullRequestTarget{
				Types: []string{
					"opened",
					"reopened",
					"ready_for_review",
					"converted_to_draft",
				},
			},
			// GITHUB_TOKEN is not used, the workflow authenticates as the GitHub App
			Permissions: ghworkflow.Permissions{},
			Jobs: map[string]*ghworkflow.Job{
				projectAutoAddWorkflowName: {
					RunsOn: ghworkflow.NewRunsOnGroupLabel(ghworkflow.GenericRunner, ""),
					// Pull requests of the bots are not tracked in the project.
					//
					// Moreover, runs triggered by Dependabot get only Dependabot secrets, so the GitHub App secrets are not available.
					If: skipBotPullRequests,
					Steps: []*ghworkflow.JobStep{
						tokenStep,
						ghworkflow.Step("add to project").
							SetUsesWithComment(
								"actions/github-script@"+config.GitHubScriptActionRef,
								"version: "+config.GitHubScriptActionVersion,
							).
							SetEnv("PROJECT_OWNER", organization).
							SetEnv("PROJECT_NUMBER", strconv.Itoa(cfg.ProjectNumber)).
							SetEnv("PROJECT_STATUS_FIELD", cfg.StatusField).
							SetEnv("PROJECT_DRAFT_STATUS", cfg.DraftStatus).
							SetEnv("PROJECT_READY_STATUS", cfg.ReadyStatus).
							SetEnv("PROJECT_ITERATION_FIELD", cfg.IterationField).
							SetWith("github-token", "${{ steps.app-token.outputs.token }}").
							SetWith("script", projectAutoAddScript),
					},
				},
			},
		},
	)

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
// intended changes are logged instead of applied. When the client is nil
// (GITHUB_TOKEN unset) and DryRun is false, GitHub API integration is
// skipped entirely.
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

	if r.EnableImmutableReleases {
		if err := r.enableImmutableReleases(client); err != nil {
			return err
		}
	}

	if err := r.removeConformWebhook(client); err != nil {
		return err
	}

	if r.DryRun {
		return nil
	}

	return r.inviteBot(client)
}

func (r *Repository) enableBranchProtection(client *github.Client) error {
	// When kres runs from a release-* branch, protect that branch; otherwise
	// protect main.
	targetBranch := r.MainBranch
	targetBranchMain := true

	if strings.HasPrefix(r.meta.CurrentBranch, "release-") {
		targetBranch = r.meta.CurrentBranch
		targetBranchMain = false
	}

	enforceContexts := r.buildEnforceContexts(nil)

	if r.DryRun {
		if r.AllowPushesToMain && targetBranchMain {
			fmt.Printf(
				"dry-run: branch protection for %s would allow direct pushes: "+
					"no required pull request reviews, no required status checks\n",
				targetBranch,
			)

			return nil
		}

		fmt.Printf(
			"dry-run: branch protection for %s would require %d contexts:\n",
			targetBranch,
			len(enforceContexts),
		)

		for _, c := range enforceContexts {
			fmt.Printf("  - %s\n", c)
		}

		return nil
	}

	return r.applyBranchProtection(client, targetBranch, targetBranchMain, enforceContexts)
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

		existing, resp, err := client.Issues.GetLabel(
			context.Background(),
			r.meta.GitHubOrganization,
			r.meta.GitHubRepository,
			name,
		)
		if err != nil {
			if resp == nil || resp.StatusCode != http.StatusNotFound {
				return fmt.Errorf("failed to check label %q: %w", name, err)
			}

			if _, _, err := client.Issues.CreateLabel(
				context.Background(),
				r.meta.GitHubOrganization,
				r.meta.GitHubRepository,
				github.CreateIssueLabelRequest{
					Name:        name,
					Color:       new(autoLabelColor),
					Description: new(desc),
				},
			); err != nil {
				return fmt.Errorf("failed to create label %q: %w", name, err)
			}

			continue
		}

		patch := github.UpdateIssueLabelRequest{}

		if desc != "" && existing.GetDescription() != desc {
			patch.Description = new(desc)
		}

		if existing.GetColor() != autoLabelColor {
			patch.Color = new(autoLabelColor)
		}

		if patch.Description != nil || patch.Color != nil {
			if _, _, err := client.Issues.UpdateLabel(
				context.Background(),
				r.meta.GitHubOrganization,
				r.meta.GitHubRepository,
				name,
				patch,
			); err != nil {
				return fmt.Errorf("failed to update label %q: %w", name, err)
			}
		}
	}

	return nil
}

// enableImmutableReleases turns on immutable releases for the repository so that published release assets and tags can no longer be altered. Existing
// releases stay mutable until republished; the setting only affects new ones.
func (r *Repository) enableImmutableReleases(client *github.Client) error {
	if r.DryRun {
		fmt.Println("dry-run: repository would enable immutable releases")

		return nil
	}

	status, _, err := client.Repositories.AreImmutableReleasesEnabled(
		context.Background(),
		r.meta.GitHubOrganization,
		r.meta.GitHubRepository,
	)
	if err != nil {
		return fmt.Errorf("failed to check immutable releases status: %w", err)
	}

	// Already enabled directly or enforced by the org policy: nothing to do.
	if status.GetEnabled() || status.GetEnforcedByOwner() {
		return nil
	}

	if _, err = client.Repositories.EnableImmutableReleases(
		context.Background(),
		r.meta.GitHubOrganization,
		r.meta.GitHubRepository,
	); err != nil {
		return fmt.Errorf("failed to enable immutable releases: %w", err)
	}

	fmt.Println("immutable releases enabled")

	return nil
}

func (r *Repository) buildEnforceContexts(base []string) []string {
	enforceContexts := base

	if r.autoContextsFunc != nil {
		enforceContexts = append(enforceContexts, r.autoContextsFunc()...)
	}

	if !r.meta.CompileGithubWorkflowsOnly {
		enforceContexts = append(enforceContexts, "default")
	}

	enforceContexts = append(enforceContexts, r.meta.ExtraEnforcedContexts...)

	if r.EnableConform {
		enforceContexts = append(enforceContexts, conformJobName)
	}

	return slices.Compact(slices.Sorted(slices.Values(enforceContexts)))
}

//nolint:gocyclo,cyclop
func (r *Repository) applyBranchProtection(
	client *github.Client,
	branch string,
	branchIsMain bool,
	enforceContexts []string,
) error {
	branchProtection, resp, err := client.Repositories.GetBranchProtection(
		context.Background(),
		r.meta.GitHubOrganization,
		r.meta.GitHubRepository,
		branch,
	)
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

	if r.AllowPushesToMain && branchIsMain {
		// Required reviews block direct pushes outright, and required status
		// checks reject any push whose commit doesn't already have them passing,
		// so both have to go for pushes to the branch to be possible.
		req.RequiredPullRequestReviews = nil
		req.RequiredStatusChecks = nil
	}

	if branchProtection != nil {
		sigProtected, _, sigErr := client.Repositories.GetSignaturesProtectedBranch(
			context.Background(),
			r.meta.GitHubOrganization,
			r.meta.GitHubRepository,
			branch,
		)
		if sigErr != nil {
			return nil //nolint:nilerr
		}

		// check if everything is already set up
		if branchProtection.GetAllowDeletions().Enabled == *req.AllowDeletions &&
			branchProtection.GetAllowForcePushes().Enabled == *req.AllowForcePushes &&
			branchProtection.GetEnforceAdmins().Enabled == req.EnforceAdmins &&
			branchProtection.GetRequireLinearHistory().Enabled == *req.RequireLinearHistory &&
			pullRequestReviewsMatch(branchProtection, req.RequiredPullRequestReviews) &&
			statusChecksMatch(branchProtection, req.RequiredStatusChecks, enforceContexts) &&
			sigProtected.GetEnabled() {
			return nil
		}
	}

	_, updateResp, err := client.Repositories.UpdateBranchProtection(
		context.Background(),
		r.meta.GitHubOrganization,
		r.meta.GitHubRepository,
		branch,
		&req,
	)
	if err != nil {
		if updateResp != nil && updateResp.StatusCode == http.StatusNotFound {
			fmt.Printf("branch %s not found, skipping protection\n", branch)

			return nil
		}

		return err
	}

	if _, _, err = client.Repositories.RequireSignaturesOnProtectedBranch(
		context.Background(),
		r.meta.GitHubOrganization,
		r.meta.GitHubRepository,
		branch,
	); err != nil {
		return err
	}

	fmt.Printf("branch protection settings updated for %s\n", branch)

	return nil
}

// removeConformWebhook removes the legacy conform webhook from the repository:
// conform now runs as a GitHub Actions workflow (see CompileGitHubWorkflow).
func (r *Repository) removeConformWebhook(client *github.Client) error {
	if r.ConformWebhookURL == "" {
		return nil
	}

	if r.DryRun {
		fmt.Printf("dry-run: repository would remove conform webhook %s (if present)\n", r.ConformWebhookURL)

		return nil
	}

	opts := &github.ListOptions{PerPage: 100}

	for {
		hooks, resp, err := client.Repositories.ListHooks(
			context.Background(),
			r.meta.GitHubOrganization,
			r.meta.GitHubRepository,
			opts,
		)
		if err != nil {
			return fmt.Errorf("failed to list webhooks: %w", err)
		}

		for _, hook := range hooks {
			if hook.GetConfig().GetURL() != r.ConformWebhookURL {
				continue
			}

			if _, err = client.Repositories.DeleteHook(
				context.Background(),
				r.meta.GitHubOrganization,
				r.meta.GitHubRepository,
				hook.GetID(),
			); err != nil {
				return fmt.Errorf("failed to delete conform webhook: %w", err)
			}

			fmt.Println("conform webhook removed")
		}

		if resp.NextPage == 0 {
			return nil
		}

		opts.Page = resp.NextPage
	}
}

func (r *Repository) inviteBot(client *github.Client) error {
	users, _, err := client.Repositories.ListCollaborators(
		context.Background(),
		r.meta.GitHubOrganization,
		r.meta.GitHubRepository,
		&github.ListCollaboratorsOptions{},
	)
	if err != nil {
		return err
	}

	for _, user := range users {
		if user.GetLogin() == r.BotName {
			return nil
		}
	}

	_, resp, err := client.Repositories.AddCollaborator(
		context.Background(),
		r.meta.GitHubOrganization,
		r.meta.GitHubRepository,
		r.BotName,
		&github.RepositoryAddCollaboratorOptions{
			Permission: "maintain",
		},
	)
	if err != nil {
		if resp.StatusCode == http.StatusNoContent {
			return nil
		}

		return err
	}

	fmt.Println("invited bot", r.BotName)

	return nil
}

// pullRequestReviewsMatch reports whether the pull request review enforcement
// on the branch already matches what is requested (including the relaxed case
// where no reviews are required at all).
func pullRequestReviewsMatch(
	branchProtection *github.Protection,
	req *github.PullRequestReviewsEnforcementRequest,
) bool {
	current := branchProtection.GetRequiredPullRequestReviews()

	if req == nil {
		return current == nil
	}

	return current != nil &&
		current.DismissStaleReviews == req.DismissStaleReviews &&
		current.RequiredApprovingReviewCount == req.RequiredApprovingReviewCount
}

// statusChecksMatch reports whether the required status checks on the branch
// already match what is requested (including the relaxed case where no checks
// are required at all).
func statusChecksMatch(
	branchProtection *github.Protection,
	req *github.RequiredStatusChecks,
	enforceContexts []string,
) bool {
	current := branchProtection.GetRequiredStatusChecks()

	if req == nil {
		return current == nil
	}

	return current != nil &&
		current.Strict == req.Strict &&
		current.Checks != nil &&
		equalStringSlices(
			xslices.Map(*current.Checks, func(s *github.RequiredStatusCheck) string {
				return s.Context
			}), enforceContexts,
		)
}

func equalStringSlices(a, b []string) bool {
	a = slices.Clone(a)
	b = slices.Clone(b)

	slices.Sort(a)
	slices.Sort(b)

	return slices.Equal(a, b)
}

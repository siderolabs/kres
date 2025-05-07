// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package helm provides helm build node.
package helm

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Build is a helm build node.
type Build struct {
	meta *meta.Options

	dag.BaseNode
}

// NewBuild initializes Build.
func NewBuild(meta *meta.Options) *Build {
	return &Build{
		meta: meta,

		BaseNode: dag.NewBaseNode("helm"),
	}
}

// CompileMakefile implements makefile.Compiler.
func (helm *Build) CompileMakefile(output *makefile.Output) error {
	helmReleaseScript := fmt.Sprintf(`@helm push $(ARTIFACTS)/%s-*.tgz oci://$(HELMREPO) 2>&1 | tee $(ARTIFACTS)/.digest
@cosign sign --yes $(COSING_ARGS) $(HELMREPO)/%s@$$(cat $(ARTIFACTS)/.digest | awk -F "[, ]+" '/Digest/{print $$NF}')
`, filepath.Base(helm.meta.HelmChartDir), filepath.Base(helm.meta.HelmChartDir))

	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable("HELMREPO", "$(REGISTRY)/$(USERNAME)/charts")).
		Variable(makefile.OverridableVariable("COSIGN_ARGS", ""))

	generateTarget := output.GetTarget("generate")
	if generateTarget != nil {
		generateTarget.Script(fmt.Sprintf(`@sed -i "s/appVersion: .*/appVersion: \"$$(cat internal/version/data/tag)\"/" %s/Chart.yaml`, helm.meta.HelmChartDir))
	}

	output.Target("helm").
		Description("Package helm chart").
		Phony().
		Script(fmt.Sprintf("@helm package %s -d $(ARTIFACTS)", helm.meta.HelmChartDir))

	output.Target("helm-release").
		Description("Release helm chart").
		Phony().
		Depends("helm").
		Script(helmReleaseScript)

	return nil
}

// CompileGitHubWorkflow implements ghworkflow.Compiler.
func (helm *Build) CompileGitHubWorkflow(output *ghworkflow.Output) error {
	cosignInstallStep := ghworkflow.Step("Install cosign").
		SetUses(fmt.Sprintf("sigstore/cosign-installer@%s", config.CosignInstallActionVerson))

	if err := cosignInstallStep.SetConditions("except-pull-request"); err != nil {
		return err
	}

	loginStep := ghworkflow.Step("Login to registry").
		SetUses("docker/login-action@"+config.LoginActionVersion).
		SetWith("registry", "ghcr.io").
		SetWith("username", "${{ github.repository_owner }}").
		SetWith("password", "${{ secrets.GITHUB_TOKEN }}")

	if err := loginStep.SetConditions("except-pull-request"); err != nil {
		return err
	}

	lintStep := ghworkflow.Step("Lint chart").
		SetCommand(fmt.Sprintf("helm lint %s", helm.meta.HelmChartDir))

	if err := lintStep.SetConditions("on-pull-request"); err != nil {
		return err
	}

	templateStep := ghworkflow.Step("Template chart").
		SetCommand(fmt.Sprintf("helm template -f %s %s %s", filepath.Join(helm.meta.HelmChartDir, "values.yaml"), filepath.Base(helm.meta.HelmChartDir), helm.meta.HelmChartDir))

	if err := templateStep.SetConditions("on-pull-request"); err != nil {
		return err
	}

	helmLoginStep := ghworkflow.Step("helm login").
		SetEnv("HELM_CONFIG_HOME", "/var/tmp/.config/helm").
		SetCommand(fmt.Sprintf("helm registry login -u %s -p ${{ secrets.GITHUB_TOKEN }} ghcr.io", "${{ github.repository_owner }}"))

	if err := helmLoginStep.SetConditions("only-on-tag"); err != nil {
		return err
	}

	helmReleaseStep := ghworkflow.Step("Release chart").
		SetEnv("HELM_CONFIG_HOME", "/var/tmp/.config/helm").
		SetMakeStep("helm-release")

	if err := helmReleaseStep.SetConditions("only-on-tag"); err != nil {
		return err
	}

	jobPermissions := ghworkflow.DefaultJobPermissions()
	jobPermissions["id-token"] = "write"

	output.AddWorkflow("helm", &ghworkflow.Workflow{
		Name: "helm",
		Concurrency: ghworkflow.Concurrency{
			Group:            "helm-${{ github.head_ref || github.run_id }}",
			CancelInProgress: true,
		},
		On: ghworkflow.On{
			Push: ghworkflow.Push{
				Tags: []string{"v*"},
			},
			PullRequest: ghworkflow.PullRequest{
				Branches: ghworkflow.Branches{
					"main",
					"release-*",
				},
				Paths: []string{fmt.Sprintf("%s/**", filepath.Dir(helm.meta.HelmChartDir))},
			},
		},
		Jobs: map[string]*ghworkflow.Job{
			"default": {
				Permissions: jobPermissions,
				RunsOn: []string{
					ghworkflow.HostedRunner,
					ghworkflow.GenericRunner,
				},
				Steps: slices.Concat(
					ghworkflow.CommonSteps(),
					[]*ghworkflow.JobStep{
						{
							Name: "Install Helm",
							Uses: fmt.Sprintf("azure/setup-helm@%s", config.HelmSetupActionVersion),
						},
						cosignInstallStep,
						loginStep,
						lintStep,
						templateStep,
						helmLoginStep,
						helmReleaseStep,
					},
				),
			},
		},
	})

	return nil
}

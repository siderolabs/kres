// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package helm provides helm build node.
package helm

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/output/dockerignore"
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
	meta.BuildArgs.Add(
		"HELMDOCS_VERSION",
	)

	return &Build{
		meta: meta,

		BaseNode: dag.NewBaseNode("helm"),
	}
}

// CompileDockerfile implements dockerfile.Compiler.
func (helm *Build) CompileDockerfile(output *dockerfile.Output) error {
	output.Stage("helm-toolchain").
		Description("helm toolchain").
		From("base").
		Step(step.Arg("HELMDOCS_VERSION")).
		Step(step.Script(
			fmt.Sprintf(
				"go install github.com/norwoodj/helm-docs/cmd/helm-docs@${HELMDOCS_VERSION} \\\n"+
					"\t&& mv /go/bin/helm-docs %s/helm-docs", helm.meta.BinPath),
		).
			MountCache(filepath.Join(helm.meta.CachePath, "go-build"), helm.meta.GitHubRepository).
			MountCache(filepath.Join(helm.meta.GoPath, "pkg"), helm.meta.GitHubRepository),
		)

	output.Stage("helm-docs-run").
		Description("runs helm-docs").
		From("helm-toolchain").
		Step(step.Copy(helm.meta.HelmChartDir, filepath.Join("/src", helm.meta.HelmChartDir))).
		Step(step.Run("helm-docs", "--badge-style=flat", "--template-files=README.md.gotpl").
			MountCache(filepath.Join(helm.meta.CachePath, "go-build"), helm.meta.GitHubRepository).
			MountCache(filepath.Join(helm.meta.CachePath, "helm-docs"), helm.meta.GitHubRepository, step.CacheLocked))

	output.Stage("helm-docs").
		Description("clean helm-docs output").
		From("scratch").
		Step(step.Copy(filepath.Join("/src", helm.meta.HelmChartDir), helm.meta.HelmChartDir).From("helm-docs-run"))

	return nil
}

// CompileDockerignore implements dockerignore.Compiler.
func (helm *Build) CompileDockerignore(output *dockerignore.Output) error {
	output.
		AllowLocalPath(helm.meta.HelmChartDir)

	return nil
}

// CompileMakefile implements makefile.Compiler.
func (helm *Build) CompileMakefile(output *makefile.Output) error {
	helmReleaseScript := fmt.Sprintf(`@helm push $(ARTIFACTS)/%s-*.tgz oci://$(HELMREPO) 2>&1 | tee $(ARTIFACTS)/.digest
@cosign sign --yes $(COSIGN_ARGS) $(HELMREPO)/%s@$$(cat $(ARTIFACTS)/.digest | awk -F "[, ]+" '/Digest/{print $$NF}')
`, filepath.Base(helm.meta.HelmChartDir), filepath.Base(helm.meta.HelmChartDir))

	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable("HELMREPO", "$(REGISTRY)/$(USERNAME)/charts")).
		Variable(makefile.OverridableVariable("COSIGN_ARGS", "")).
		Variable(makefile.OverridableVariable("HELMDOCS_VERSION", config.HelmDocsVersion))

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

	output.Target("chart-lint").
		Description("Lint helm chart").
		Phony().
		Script(fmt.Sprintf("@helm lint %s", helm.meta.HelmChartDir))

	output.Target("helm-plugin-install").
		Description("Install helm plugins").
		Phony().
		Script(
			fmt.Sprintf("-helm plugin install https://github.com/helm-unittest/helm-unittest.git --verify=false --version=%s", config.HelmUnitTestVersion),
			fmt.Sprintf("-helm plugin install https://github.com/losisin/helm-values-schema-json.git --verify=false --version=%s", config.HelmValuesSchemaJSONVersion),
		)

	output.Target("kuttl-plugin-install").
		Description("Install kubectl kuttl plugin").
		Phony().
		Script("kubectl krew install kuttl")

	output.Target("chart-e2e").
		Description("Run helm chart e2e tests").
		Phony().
		Script(fmt.Sprintf("export KUBECONFIG=$(shell pwd)/$(ARTIFACTS)/kubeconfig && cd %s && kubectl kuttl test", helm.meta.HelmE2EDir))

	output.Target("chart-unittest").
		Description("Run helm chart unit tests").
		Phony().
		Depends("$(ARTIFACTS)").
		Script(fmt.Sprintf("@helm unittest %s --output-type junit --output-file $(ARTIFACTS)/helm-unittest-report.xml", helm.meta.HelmChartDir))

	output.Target("chart-gen-schema").
		Description("Generate helm chart schema").
		Phony().
		Script(fmt.Sprintf("@helm schema --use-helm-docs --draft=7 --indent=2 --values=%s/values.yaml --output=%s/values.schema.json", helm.meta.HelmChartDir, helm.meta.HelmChartDir))

	output.Target("helm-docs").Description("Runs helm-docs and generates chart documentation").
		Phony().
		Script("@$(MAKE) local-$@ DEST=.")

	return nil
}

// CompileGitHubWorkflow implements ghworkflow.Compiler.
func (helm *Build) CompileGitHubWorkflow(output *ghworkflow.Output) error {
	cosignInstallStep := ghworkflow.Step("Install cosign").
		SetUsesWithComment(
			fmt.Sprintf("sigstore/cosign-installer@%s", config.CosignInstallActionRef),
			"version: "+config.CosignInstallActionVersion,
		)

	if err := cosignInstallStep.SetConditions("except-pull-request"); err != nil {
		return err
	}

	loginStep := ghworkflow.Step("Login to registry").
		SetUsesWithComment(
			"docker/login-action@"+config.LoginActionRef,
			"version: "+config.LoginActionVersion,
		).
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
		SetCommand(fmt.Sprintf("helm template -f %s %s %s %s",
			filepath.Join(helm.meta.HelmChartDir, "values.yaml"),
			strings.Join(helm.meta.HelmTemplateFlags, " "),
			filepath.Base(helm.meta.HelmChartDir),
			helm.meta.HelmChartDir,
		))

	if err := templateStep.SetConditions("on-pull-request"); err != nil {
		return err
	}

	unittestPluginInstallStep := ghworkflow.Step("Install unit test plugin").
		SetMakeStep("helm-plugin-install")

	if err := unittestPluginInstallStep.SetConditions("on-pull-request"); err != nil {
		return err
	}

	unittestStep := ghworkflow.Step("Unit test chart").
		SetMakeStep("chart-unittest")

	if err := unittestStep.SetConditions("on-pull-request"); err != nil {
		return err
	}

	schemaStep := ghworkflow.Step("Generate schema").
		SetMakeStep("chart-gen-schema")

	if err := schemaStep.SetConditions("on-pull-request"); err != nil {
		return err
	}

	docsStep := ghworkflow.Step("Generate docs").
		SetMakeStep("helm-docs")

	if err := docsStep.SetConditions("on-pull-request"); err != nil {
		return err
	}

	checkDirtyStep := ghworkflow.Step("Check dirty").
		SetMakeStep("check-dirty")

	if err := checkDirtyStep.SetConditions("on-pull-request"); err != nil {
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
			ghworkflow.DefaultJobName: {
				Permissions: jobPermissions,
				RunsOn:      ghworkflow.NewRunsOnGroupLabel(ghworkflow.GenericRunner, ""),
				Steps: slices.Concat(
					ghworkflow.CommonSteps(),
					[]*ghworkflow.JobStep{
						{
							Name: "Install Helm",
							Uses: ghworkflow.ActionRef{
								Image:   fmt.Sprintf("azure/setup-helm@%s", config.HelmSetupActionRef),
								Comment: "version: " + config.HelmSetupActionVersion,
							},
						},
						cosignInstallStep,
						loginStep,
						lintStep,
						templateStep,
						unittestPluginInstallStep,
						unittestStep,
						schemaStep,
						docsStep,
						checkDirtyStep,
						helmLoginStep,
						helmReleaseStep,
					},
				),
			},
		},
	})

	return nil
}

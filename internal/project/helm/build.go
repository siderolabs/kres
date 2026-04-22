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
		From("--platform=${BUILDPLATFORM} ${TOOLCHAIN}").
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
		Step(step.WorkDir("/src")).
		Step(step.Copy(helm.meta.HelmChartDir, filepath.Join("/src", helm.meta.HelmChartDir))).
		Step(step.Run("helm-docs", "--badge-style=flat").
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

	var hasGenerate bool

	for _, parent := range helm.Parents() {
		if dag.FindByName("protobuf", parent.Inputs()...) != nil {
			hasGenerate = true

			break
		}
	}

	if hasGenerate {
		generateTarget := output.Target("generate")
		generateTarget.Depends("helm-plugin-install")

		// Only update Chart.yaml for final releases (vX.Y.Z, no pre-release suffix).
		// This prevents dirty tags, dev builds, and pre-releases from polluting the chart.
		if helm.meta.ChartVersionMajor != nil {
			// The chart version mirrors the app's minor.patch with the configured
			// major, e.g. app v1.5.9 -> chart 2.5.9.
			generateTarget.Script(fmt.Sprintf(`@TAG=$$(cat internal/version/data/tag); \
if echo "$$TAG" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$$'; then \
  sed -i "s/^appVersion: .*/appVersion: \"$$TAG\"/" %[1]s/Chart.yaml; \
  MINOR_PATCH=$$(echo "$$TAG" | sed 's/^v[0-9]*\.//'); \
  sed -i "s/^version: .*/version: %[2]d.$$MINOR_PATCH/" %[1]s/Chart.yaml; \
fi`, helm.meta.HelmChartDir, *helm.meta.ChartVersionMajor))
		} else {
			generateTarget.Script(fmt.Sprintf(`@TAG=$$(cat internal/version/data/tag); \
if echo "$$TAG" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$$'; then \
  sed -i "s/^appVersion: .*/appVersion: \"$$TAG\"/" %[1]s/Chart.yaml; \
fi`, helm.meta.HelmChartDir))
		}

		// Regenerate helm docs and schema as part of generate, so check-dirty passes in CI.
		if helm.meta.EnforceHelmDocs {
			generateTarget.Script("@$(MAKE) helm-docs")
		}

		if helm.meta.EnforceHelmSchema {
			generateTarget.Script("@$(MAKE) chart-gen-schema")
		}
	}

	output.Target("helm").
		Description("Package helm chart").
		Phony().
		Depends("$(ARTIFACTS)").
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
		SetCommand(fmt.Sprintf("helm template %s %s %s",
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

	// When chartVersionMajor is set, only release stable (non-prerelease) charts.
	releaseCondition := "only-on-tag"
	if helm.meta.ChartVersionMajor != nil {
		releaseCondition = "only-on-stable-tag"
	}

	helmLoginStep := ghworkflow.Step("helm login").
		SetEnv("HELM_CONFIG_HOME", "/var/tmp/.config/helm").
		SetCommand(fmt.Sprintf("helm registry login -u %s -p ${{ secrets.GITHUB_TOKEN }} ghcr.io", "${{ github.repository_owner }}"))

	if err := helmLoginStep.SetConditions(releaseCondition); err != nil {
		return err
	}

	helmReleaseStep := ghworkflow.Step("Release chart").
		SetEnv("HELM_CONFIG_HOME", "/var/tmp/.config/helm").
		SetMakeStep("helm-release")

	if err := helmReleaseStep.SetConditions(releaseCondition); err != nil {
		return err
	}

	jobPermissions := ghworkflow.DefaultJobPermissions()
	jobPermissions["id-token"] = "write"

	jobSteps := []*ghworkflow.JobStep{
		ghworkflow.SetupBuildxStep(),
		loginStep,
		lintStep,
		templateStep,
	}

	// Add steps for unit tests
	jobSteps = append(jobSteps, []*ghworkflow.JobStep{unittestPluginInstallStep, unittestStep}...)

	// Add steps for schema generation and docs generation if enforced
	if helm.meta.EnforceHelmSchema || helm.meta.EnforceHelmDocs {
		var helmSteps []*ghworkflow.JobStep

		if helm.meta.EnforceHelmSchema {
			helmSteps = append(helmSteps, schemaStep)
		}

		if helm.meta.EnforceHelmDocs {
			helmSteps = append(helmSteps, docsStep)
		}

		jobSteps = append(jobSteps, helmSteps...)
	}

	// Add final steps
	jobSteps = append(jobSteps, []*ghworkflow.JobStep{helmLoginStep, helmReleaseStep}...)

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
					jobSteps,
				),
			},
		},
	})

	return nil
}

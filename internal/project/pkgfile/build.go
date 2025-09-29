// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package pkgfile provides building blocks for Pkgfile.
package pkgfile

import (
	"fmt"
	"slices"
	"strings"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerignore"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/output/renovate"
	"github.com/siderolabs/kres/internal/project/meta"
)

const (
	renovateMatchStringPkgfile = `# renovate: datasource=(?<datasource>.*?)(?:\s+extractVersion=(?<extractVersion>.+?))?(?:\s+versioning=(?<versioning>.+?))?\s+depName=(?<depName>.+?)?\s(?:.*_(?:version|VERSION):\s+(?<currentValue>.*))?(?:(\s)?.*_(?:ref|REF):\s+(?<currentDigest>.*))?` //nolint:lll
)

// Build provides common pkgfile build environment settings.
type Build struct {
	dag.BaseNode

	meta *meta.Options

	ReproducibleTargetName string              `yaml:"reproducibleTargetName"`
	AdditionalTargets      map[string][]string `yaml:"additionalTargets"`
	Targets                []string            `yaml:"targets"`
	ExtraBuildArgs         []string            `yaml:"extraBuildArgs"`
	Makefile               struct {
		ExtraVariables []struct {
			Name         string `yaml:"name"`
			DefaultValue string `yaml:"defaultValue"`
		} `yaml:"extraVariables"`
	} `yaml:"makefile"`
	UseBldrPkgTagResolver bool `yaml:"useBldrPkgTagResolver"`
}

var (
	reproducibilityTestScript = `
@rm -rf $(ARTIFACTS)/build-a $(ARTIFACTS)/build-b
@$(MAKE) local-$* DEST=$(ARTIFACTS)/build-a
@$(MAKE) local-$* DEST=$(ARTIFACTS)/build-b TARGET_ARGS="--no-cache"
@touch -ch -t $$(date -d @$(SOURCE_DATE_EPOCH) +%Y%m%d0000) $(ARTIFACTS)/build-a $(ARTIFACTS)/build-b
@diffoscope $(ARTIFACTS)/build-a $(ARTIFACTS)/build-b
@rm -rf $(ARTIFACTS)/build-a $(ARTIFACTS)/build-b
`

	bldrDownloadScript = `
@curl -sSL https://github.com/siderolabs/bldr/releases/download/$(BLDR_RELEASE)/bldr-$(OPERATING_SYSTEM)-$(GOARCH) -o $(ARTIFACTS)/bldr
@chmod +x $(ARTIFACTS)/bldr
`
)

// NewBuild initializes Build.
func NewBuild(meta *meta.Options) *Build {
	return &Build{
		meta: meta,

		BaseNode: dag.NewBaseNode("pkgfile"),
	}
}

// CompileDockerignore implements dockerignore.Compiler.
func (pkgfile *Build) CompileDockerignore(output *dockerignore.Output) error {
	output.AllowLocalPath("pkg.yaml")

	return nil
}

// CompileMakefile implements makefile.Compiler.
func (pkgfile *Build) CompileMakefile(output *makefile.Output) error {
	output.VariableGroup(makefile.VariableGroupSourceDateEpoch).
		Variable(makefile.SimpleVariable("INITIAL_COMMIT_SHA", "$(shell git rev-list --max-parents=0 HEAD)")).
		Variable(makefile.SimpleVariable("SOURCE_DATE_EPOCH", "$(shell git log $(INITIAL_COMMIT_SHA) --pretty=%ct)"))

	output.VariableGroup("sync bldr image with pkgfile").
		Variable(makefile.SimpleVariable("BLDR_RELEASE", config.BldrImageVersion)).
		Variable(makefile.SimpleVariable("BLDR_IMAGE", "ghcr.io/siderolabs/bldr:$(BLDR_RELEASE)")).
		Variable(makefile.SimpleVariable("BLDR", "docker run --rm --user $(shell id -u):$(shell id -g) --volume $(PWD):/src --entrypoint=/bldr $(BLDR_IMAGE) --root=/src"))

	buildArgs := makefile.RecursiveVariable("BUILD_ARGS", "--build-arg=SOURCE_DATE_EPOCH=$(SOURCE_DATE_EPOCH)")

	for _, arg := range pkgfile.ExtraBuildArgs {
		buildArgs.Push(fmt.Sprintf("--build-arg=%s=\"$(%s)\"", arg, arg))
	}

	commonArgs := makefile.RecursiveVariable("COMMON_ARGS", "--file=Pkgfile").
		Push("--provenance=false").
		Push("--progress=$(PROGRESS)").
		Push("--platform=$(PLATFORM)").
		Push("$(BUILD_ARGS)")

	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable("REGISTRY", "ghcr.io")).
		Variable(makefile.OverridableVariable("USERNAME", strings.ToLower(pkgfile.meta.GitHubOrganization))).
		Variable(makefile.OverridableVariable("REGISTRY_AND_USERNAME", "$(REGISTRY)/$(USERNAME)"))

	output.VariableGroup(makefile.VariableGroupDocker).
		Variable(makefile.SimpleVariable("BUILD", "docker buildx build")).
		Variable(makefile.OverridableVariable("PLATFORM", "linux/amd64,linux/arm64")).
		Variable(makefile.OverridableVariable("PROGRESS", "auto")).
		Variable(makefile.OverridableVariable("PUSH", "false")).
		Variable(makefile.OverridableVariable("CI_ARGS", "")).
		Variable(buildArgs).
		Variable(commonArgs)

	for _, arg := range pkgfile.Makefile.ExtraVariables {
		output.VariableGroup(makefile.VariableGroupExtra).
			Variable(makefile.OverridableVariable(arg.Name, arg.DefaultValue))
	}

	output.Target("target-%").
		Description("Builds the specified target defined in the Pkgfile. The build result will only remain in the build cache.").
		Script(`@$(BUILD) --target=$* $(COMMON_ARGS) $(TARGET_ARGS) $(CI_ARGS) .`)

	output.Target("local-%").
		Description("Builds the specified target defined in the Pkgfile using the local output type. The build result will be output to the specified local destination.").
		Script(`@$(MAKE) target-$* TARGET_ARGS="--output=type=local,dest=$(DEST) $(TARGET_ARGS)"`)

	output.Target("docker-%").
		Description("Builds the specified target defined in the Pkgfile using the docker output type. The build result will be loaded into Docker.").
		Script(`@$(MAKE) target-$* TARGET_ARGS="$(TARGET_ARGS)"`)

	if pkgfile.ReproducibleTargetName != "" {
		output.Target("reproducibility-test").
			Description("Builds the reproducibility test target").
			Script("@$(MAKE) reproducibility-test-local-" + pkgfile.ReproducibleTargetName)
	}

	output.Target("reproducibility-test-local-%").
		Description("Builds the specified target defined in the Pkgfile using the local output type with and without cahce. The build result will be output to the specified local destination").
		Script(reproducibilityTestScript)

	output.Target("$(ARTIFACTS)/bldr").
		Description("Downloads bldr binary.").
		Script(bldrDownloadScript).
		Depends("$(ARTIFACTS)")

	output.Target("update-checksums").
		Depends("$(ARTIFACTS)/bldr").
		Phony().
		Description("Updates the checksums in the Pkgfile/vars.yaml based on the changed version variables.").
		Script(`@git diff -U0 | $(ARTIFACTS)/bldr update`)

	output.VariableGroup(makefile.VariableGroupTargets).
		Variable(makefile.RecursiveVariable("TARGETS", strings.Join(pkgfile.Targets, "\n")))

	output.Target("all").
		Description("Builds all targets defined.").
		Depends("$(TARGETS)")

	defaultTarget := "$(TARGETS)"

	for name, targets := range pkgfile.AdditionalTargets {
		targetName := strings.ToUpper(name) + "_TARGETS"
		targetNameVariable := fmt.Sprintf("$(%s)", targetName)
		defaultTarget += " " + targetNameVariable

		output.VariableGroup(makefile.VariableGroupTargets).
			Variable(makefile.RecursiveVariable(targetName, strings.Join(targets, "\n")))

		output.Target(name).
			Description(fmt.Sprintf("Builds all %s targets defined.", name)).
			Depends(targetNameVariable)
	}

	if pkgfile.UseBldrPkgTagResolver {
		output.Target(defaultTarget).
			Script("@$(MAKE) docker-$@ TARGET_ARGS=\"--tag=$(REGISTRY)/$(USERNAME)/$@:$(shell $(ARTIFACTS)/bldr eval --target $@ --build-arg TAG=$(TAG) '{{.VERSION}}' 2>/dev/null) --push=$(PUSH)\"").
			Phony().
			Depends("$(ARTIFACTS)/bldr")
	} else {
		output.Target(defaultTarget).
			Script("@$(MAKE) docker-$@ TARGET_ARGS=\"--tag=$(REGISTRY_AND_USERNAME)/$@:$(TAG) --push=$(PUSH)\"").
			Phony()
	}

	output.Target("deps.svg").
		Description("Generates a dependency graph of the Pkgfile.").
		Script(`@rm -f deps.png`).
		Script(`@$(BLDR) graph $(BUILD_ARGS) | dot -Tsvg -o deps.svg`).
		Phony()

	return nil
}

// CompileGitHubWorkflow implements ghworkflow.Compiler.
func (pkgfile *Build) CompileGitHubWorkflow(output *ghworkflow.Output) error {
	output.SetOptionsForPkgs()

	loginStep := ghworkflow.Step("Login to registry").
		SetUses("docker/login-action@"+config.LoginActionVersion).
		SetWith("registry", "ghcr.io").
		SetWith("username", "${{ github.repository_owner }}").
		SetWith("password", "${{ secrets.GITHUB_TOKEN }}")

	if err := loginStep.SetConditions("except-pull-request"); err != nil {
		return err
	}

	pushStep := ghworkflow.Step("Push to registry").SetMakeStep("", "PUSH=true")

	if err := pushStep.SetConditions("except-pull-request"); err != nil {
		return err
	}

	buildStep := ghworkflow.Step("Build").SetMakeStep("")

	if err := buildStep.SetConditions("on-pull-request"); err != nil {
		return err
	}

	output.AddStep(
		"default",
		buildStep,
	)

	steps := []*ghworkflow.JobStep{
		loginStep,
		pushStep,
	}

	for name := range pkgfile.AdditionalTargets {
		buildStep := ghworkflow.Step("Build " + name).SetMakeStep(name)

		if err := buildStep.SetConditions("on-pull-request"); err != nil {
			return err
		}

		output.AddStep(
			"default",
			buildStep,
		)

		pushStep := ghworkflow.Step("Push "+name).SetMakeStep(name, "PUSH=true")

		if err := pushStep.SetConditions("except-pull-request"); err != nil {
			return err
		}

		steps = append(
			steps,
			pushStep,
		)
	}

	output.AddStep("default", steps...)

	if pkgfile.ReproducibleTargetName != "" {
		output.AddStep(
			"default",
			ghworkflow.Step("Retrieve PR labels").
				SetID("retrieve-pr-labels").
				SetUses("actions/github-script@"+config.GitHubScriptActionVersion).
				SetWith("retries", "3").
				SetWith("script", strings.TrimPrefix(ghworkflow.IssueLabelRetrieveScript, "\n")),
		)

		output.AddOutputs("default", map[string]string{
			"labels": "${{ steps.retrieve-pr-labels.outputs.result }}",
		})

		output.AddJob("reproducibility", false, &ghworkflow.Job{
			RunsOn: ghworkflow.NewRunsOnGroupLabel(ghworkflow.PkgsRunner, ""),
			If:     "contains(fromJSON(needs.default.outputs.labels), 'integration/reproducibility')",
			Needs:  []string{"default"},
			Steps:  ghworkflow.DefaultPkgsSteps(),
		})
		output.AddStep("reproducibility", ghworkflow.Step("reproducibility-test").SetMakeStep("reproducibility-test"))

		output.AddSlackNotify("weekly")
		output.AddWorkflow(
			"weekly",
			&ghworkflow.Workflow{
				Name: "weekly",
				// https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#example-using-a-fallback-value
				Concurrency: ghworkflow.Concurrency{
					Group:            "${{ github.head_ref || github.run_id }}",
					CancelInProgress: true,
				},
				On: ghworkflow.On{
					Schedule: []ghworkflow.Schedule{
						{
							Cron: "30 1 * * 1",
						},
					},
				},
				Jobs: map[string]*ghworkflow.Job{
					"reproducibility": {
						RunsOn: ghworkflow.NewRunsOnGroupLabel(ghworkflow.PkgsRunner, ""),
						Steps: append(
							ghworkflow.DefaultPkgsSteps(),
							ghworkflow.Step("reproducibility-test").SetMakeStep("reproducibility-test"),
						),
					},
				},
			},
		)
	}

	return nil
}

// CompileRenovate implements renovate.Compiler.
func (pkgfile *Build) CompileRenovate(output *renovate.Output) error {
	customManagers := []renovate.CustomManager{
		{
			CustomType:          "regex",
			ManagerFilePatterns: []string{"/Pkgfile/"},
			MatchStrings: []string{
				renovateMatchStringPkgfile,
			},
			VersioningTemplate: "{{#if versioning}}{{versioning}}{{else}}semver{{/if}}",
		},
		{
			CustomType:          "regex",
			ManagerFilePatterns: []string{"/Pkgfile/"},
			MatchStrings: []string{
				"ghcr.io\\/siderolabs\\/bldr:(?<currentValue>v.*)",
			},
			DataSourceTemplate: "github-tags",
			DepNameTemplate:    "siderolabs/bldr",
			VersioningTemplate: "semver",
		},
	}

	if pkgfile.UseBldrPkgTagResolver {
		customManagers = slices.Concat(customManagers, []renovate.CustomManager{
			{
				CustomType:          "regex",
				ManagerFilePatterns: []string{"/vars.yaml/"},
				MatchStrings: []string{
					renovateMatchStringPkgfile,
				},
				VersioningTemplate: "{{#if versioning}}{{versioning}}{{else}}semver{{/if}}",
			},
		})
	}

	output.CustomManagers(customManagers)

	return nil
}

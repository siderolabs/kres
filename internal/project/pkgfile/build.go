// This Source Code Form isasdpy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package pkgfile provides building blocks for Pkgfile.
package pkgfile

import (
	"fmt"
	"strings"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerignore"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Build provides common pkgfile build environment settings.
type Build struct {
	dag.BaseNode

	meta *meta.Options

	ReproducibleTargetName string              `yaml:"reproducibleTargetName"`
	AdditionalTargets      map[string][]string `yaml:"additionalTargets"`
	Targets                []string            `yaml:"targets"`
	ExtraBuildArgs         []string            `yaml:"extraBuildArgs"`
	UseBldrPkgTagResolver  bool                `yaml:"useBldrPkgTagResolver"`
}

var (
	goarchDef = `
$(shell uname -m | tr '[:upper:]' '[:lower:]')

ifeq ($(GOARCH),x86_64)
  GOARCH := amd64
endif
`

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
	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.SimpleVariable("OPERATING_SYSTEM", "$(shell uname -s | tr '[:upper:]' '[:lower:]')")).
		Variable(makefile.SimpleVariable("GOARCH", strings.TrimPrefix(goarchDef, "\n")))

	output.VariableGroup(makefile.VariableGroupSourceDateEpoch).
		Variable(makefile.SimpleVariable("INITIAL_COMMIT_SHA", "$(shell git rev-list --max-parents=0 HEAD)")).
		Variable(makefile.SimpleVariable("SOURCE_DATE_EPOCH", "$(shell git log $(INITIAL_COMMIT_SHA) --pretty=%ct)"))

	output.VariableGroup("sync bldr image with pkgfile").
		Variable(makefile.SimpleVariable("BLDR_RELEASE", config.BldrImageVersion)).
		Variable(makefile.SimpleVariable("BLDR_IMAGE", "ghcr.io/siderolabs/bldr:$(BLDR_RELEASE)")).
		Variable(makefile.SimpleVariable("BLDR", "docker run --rm --user $(shell id -u):$(shell id -g) --volume $(PWD):/src --entrypoint=/bldr $(BLDR_IMAGE) --root=/src"))

	buildArgs := makefile.RecursiveVariable("COMMON_ARGS", "--file=Pkgfile").
		Push("--provenance=false").
		Push("--progress=$(PROGRESS)").
		Push("--platform=$(PLATFORM)").
		Push("--build-arg=SOURCE_DATE_EPOCH=$(SOURCE_DATE_EPOCH)")

	for _, arg := range pkgfile.ExtraBuildArgs {
		buildArgs.Push(fmt.Sprintf("--build-arg=%s=\"$(%s)\"", arg, arg))
	}

	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable("REGISTRY", "ghcr.io")).
		Variable(makefile.OverridableVariable("USERNAME", pkgfile.meta.GitHubOrganization)).
		Variable(makefile.OverridableVariable("REGISTRY_AND_USERNAME", "$(REGISTRY)/$(USERNAME)"))

	output.VariableGroup(makefile.VariableGroupDocker).
		Variable(makefile.SimpleVariable("BUILD", "docker buildx build")).
		Variable(makefile.OverridableVariable("PLATFORM", "linux/amd64,linux/arm64")).
		Variable(makefile.OverridableVariable("PROGRESS", "auto")).
		Variable(makefile.OverridableVariable("PUSH", "false")).
		Variable(makefile.OverridableVariable("CI_ARGS", "")).
		Variable(buildArgs)

	output.Target("$(ARTIFACTS)").
		Description("Creates artifacts directory.").
		Script("@mkdir -p $(ARTIFACTS)")

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

		output.Target("$(ARTIFACTS)/bldr").
			Description("Downloads bldr binary.").
			Script(bldrDownloadScript).
			Depends("$(ARTIFACTS)")
	} else {
		output.Target(defaultTarget).
			Script("@$(MAKE) docker-$@ TARGET_ARGS=\"--tag=$(REGISTRY_AND_USERNAME)/$@:$(TAG) --push=$(PUSH)\"").
			Phony()
	}

	output.Target("deps.png").
		Description("Generates a dependency graph of the Pkgfile.").
		Script(`@$(BLDR) graph | dot -Tpng -o deps.png`).
		Phony()

	return nil
}

// CompileGitHubWorkflow implements ghworkflow.Compiler.
func (pkgfile *Build) CompileGitHubWorkflow(output *ghworkflow.Output) error {
	output.SetDefaultJobRunnerAsPkgs()
	output.OverwriteDefaultJobStepsAsPkgs()

	loginStep := &ghworkflow.Step{
		Name: "Login to registry",
		Uses: "docker/login-action@" + config.LoginActionVersion,
		With: map[string]string{
			"registry": "ghcr.io",
			"username": "${{ github.repository_owner }}",
			"password": "${{ secrets.GITHUB_TOKEN }}",
		},
	}

	loginStep.ExceptPullRequest()

	pushStep := ghworkflow.MakeStep("", "PUSH=true").
		SetName("Push to registry").
		ExceptPullRequest()

	output.AddStep(
		"default",
		ghworkflow.MakeStep("").SetName("Build"),
	)

	steps := []*ghworkflow.Step{
		loginStep,
		pushStep,
	}

	for name := range pkgfile.AdditionalTargets {
		output.AddStep(
			"default",
			ghworkflow.MakeStep(name).SetName("Build "+name),
		)

		steps = append(
			steps,
			ghworkflow.MakeStep(name, "PUSH=true").
				SetName("Push "+name).
				ExceptPullRequest(),
		)
	}

	output.AddStep("default", steps...)

	if pkgfile.ReproducibleTargetName != "" {
		output.AddStep(
			"default",
			&ghworkflow.Step{
				Name: "Retrieve PR labels",
				ID:   "retrieve-pr-labels",
				Uses: "actions/github-script@" + config.GitHubScriptActionVersion,
				With: map[string]string{
					"retries": "3",
					"script":  strings.TrimPrefix(ghworkflow.IssueLabelRetrieveScript, "\n"),
				},
			},
		)
		output.AddOutputs("default", map[string]string{
			"labels": "${{ steps.retrieve-pr-labels.outputs.result }}",
		})

		runnerLabels := []string{
			ghworkflow.HostedRunner,
			ghworkflow.PkgsRunner,
		}

		output.AddJob("reproducibility", &ghworkflow.Job{
			RunsOn:   runnerLabels,
			If:       "contains(fromJSON(needs.default.outputs.labels), 'integration/reproducibility')",
			Needs:    []string{"default"},
			Services: ghworkflow.DefaultServices(),
			Steps:    ghworkflow.DefaultPkgsSteps(),
		})
		output.AddStep("reproducibility", ghworkflow.MakeStep("reproducibility-test"))

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
						RunsOn:   runnerLabels,
						Services: ghworkflow.DefaultServices(),
						Steps: append(
							ghworkflow.DefaultPkgsSteps(),
							ghworkflow.MakeStep("reproducibility-test"),
						),
					},
				},
			},
		)
	}

	return nil
}

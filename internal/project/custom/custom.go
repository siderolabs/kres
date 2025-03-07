// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package custom provides blocks defined in the config manually.
package custom

import (
	"fmt"
	"strings"

	"github.com/siderolabs/gen/xslices"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	dockerstep "github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/output/drone"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Step is defined in the config manually.
//
//nolint:govet
type Step struct {
	dag.BaseNode

	meta *meta.Options

	Docker struct {
		Stages []struct {
			Name        string `yaml:"name"`
			Description string `yaml:"description"`
			From        string `yaml:"from"`
			Platform    string `yaml:"platform"`
			Workdir     string `yaml:"workdir"`
			Steps       []struct {
				Script *struct {
					Command string   `yaml:"command"`
					Cache   []string `yaml:"cache"`
				} `yaml:"script"`
				Copy *struct {
					From     string `yaml:"from"`
					Platform string `yaml:"platform"`
					Src      string `yaml:"src"`
					Dst      string `yaml:"dst"`
				} `yaml:"copy"`
				Arg string `yaml:"arg"`
			} `yaml:"steps"`
		} `yaml:"stages"`

		Enabled bool `yaml:"enabled"`
	} `yaml:"docker"`

	Makefile struct { //nolint:govet
		Enabled bool     `yaml:"enabled"`
		Phony   bool     `yaml:"phony"`
		Depends []string `yaml:"depends"`
		Script  []string `yaml:"script"`

		Variables []struct {
			Name         string `yaml:"name"`
			DefaultValue string `yaml:"defaultValue"`
		} `yaml:"variables"`
	} `yaml:"makefile"`

	Drone struct {
		Enabled     bool              `yaml:"enabled"`
		Privileged  bool              `yaml:"privileged"`
		Environment map[string]string `yaml:"environment"`
		Requests    *struct {
			CPUCores  int `yaml:"cpuCores"`
			MemoryGiB int `yaml:"memoryGiB"`
		} `yaml:"requests"`
		Volumes []struct {
			Name      string `yaml:"name"`
			MountPath string `yaml:"mountPath"`
		} `yaml:"volumes"`
		Pipelines []struct {
			Name                string            `yaml:"name"`
			Triggers            []string          `yaml:"triggers"`
			Crons               []string          `yaml:"crons"`
			EnvironmentOverride map[string]string `yaml:"environmentOverride"`
		}
	} `yaml:"drone"`

	GHAction struct {
		Environment map[string]string `yaml:"environment"`
		Condition   string            `yaml:"condition"`
		Jobs        []struct {
			Name                string            `yaml:"name"`
			EnvironmentOverride map[string]string `yaml:"environmentOverride"`
			Crons               []string          `yaml:"crons"`
			RunnerLabels        []string          `yaml:"runnerLabels"`
			TriggerLabels       []string          `yaml:"triggerLabels"`
		} `yaml:"jobs"`
		Artifacts struct {
			ExtraPaths []string `yaml:"extraPaths"`
			Additional []struct {
				Name            string   `yaml:"name"`
				Paths           []string `yaml:"paths"`
				Always          bool     `yaml:"always"`
				ContinueOnError bool     `yaml:"continueOnError"`
				RetentionDays   string   `yaml:"retentionDays"`
			} `yaml:"additional"`
			Enabled         bool   `yaml:"enabled"`
			ContinueOnError bool   `yaml:"continueOnError"`
			RetentionDays   string `yaml:"retentionDays"`
		} `yaml:"artifacts"`
		Enabled bool `yaml:"enabled"`
		SOPS    bool `yaml:"sops"`
	} `yaml:"ghaction"`

	SudoInCI bool `yaml:"sudoInCI"`
}

// NewStep initializes Step.
func NewStep(meta *meta.Options, name string) *Step {
	return &Step{
		BaseNode: dag.NewBaseNode(name),
		meta:     meta,
	}
}

// CompileDockerfile implements dockerfile.Compiler.
func (step *Step) CompileDockerfile(output *dockerfile.Output) error {
	if !step.Docker.Enabled {
		return nil
	}

	for _, stage := range step.Docker.Stages {
		s := output.Stage(stage.Name).Description(stage.Description)

		if stage.From != "" {
			s.From(stage.From)
		} else {
			s.From("scratch")
		}

		if stage.Platform != "" {
			s.Platform(stage.Platform)
		}

		if stage.Workdir != "" {
			s.Workdir(stage.Workdir)
		}

		for _, stageStep := range stage.Steps {
			switch {
			case stageStep.Arg != "":
				s.Step(dockerstep.Arg(stageStep.Arg))
			case stageStep.Script != nil:
				script := dockerstep.Script(stageStep.Script.Command)
				for _, cache := range stageStep.Script.Cache {
					script.MountCache(cache, step.meta.GitHubRepository)
				}

				s.Step(script)
			case stageStep.Copy != nil:
				copyStep := dockerstep.Copy(stageStep.Copy.Src, stageStep.Copy.Dst)
				if stageStep.Copy.From != "" {
					copyStep.From(stageStep.Copy.From)
				}

				if stageStep.Copy.Platform != "" {
					copyStep.Platform(stageStep.Copy.Platform)
				}

				s.Step(copyStep)
			}
		}
	}

	return nil
}

// CompileDrone implements drone.Compiler.
func (step *Step) CompileDrone(output *drone.Output) error {
	if !step.Drone.Enabled {
		return nil
	}

	droneMatches := func(node dag.Node) bool {
		if !dag.Implements[drone.Compiler]()(node) {
			return false
		}

		if nodeStep, ok := node.(*Step); ok {
			return nodeStep.Drone.Enabled
		}

		return true
	}

	baseDroneStep := func() *drone.Step {
		droneStep := drone.MakeStep(step.Name())

		if step.Drone.Privileged {
			droneStep.Privileged()
		}

		if step.Drone.Requests != nil {
			droneStep.ResourceRequests(step.Drone.Requests.CPUCores, step.Drone.Requests.MemoryGiB)
		}

		for k, v := range step.Drone.Environment {
			droneStep.Environment(k, v)
		}

		for _, volume := range step.Drone.Volumes {
			droneStep.EmptyDirVolume(volume.Name, volume.MountPath)
		}

		return droneStep
	}

	output.Step(baseDroneStep().
		DependsOn(dag.GatherMatchingInputNames(step, droneMatches)...))

	for _, customPipeline := range step.Drone.Pipelines {
		pipeline := output.Pipeline(customPipeline.Name, customPipeline.Triggers, customPipeline.Crons)

		type baseDroneStepper interface {
			BuildBaseDroneSteps(output drone.StepService)
		}

		var baseStepNames []string

		for _, baseInput := range dag.GatherMatchingInputsRecursive(step, dag.Implements[baseDroneStepper]()) {
			baseInput.(baseDroneStepper).BuildBaseDroneSteps(pipeline) //nolint:forcetypeassert,errcheck // type is checked in GatherMatchingInputsRecursive

			baseStepNames = append(baseStepNames, baseInput.Name())
		}

		pipeline.Step(
			drone.CustomStep("load-artifacts",
				`az login --service-principal -u "$${AZURE_STORAGE_USER}" -p "$${AZURE_STORAGE_PASS}" --tenant "$${AZURE_TENANT}"`,
				`mkdir -p `+step.meta.ArtifactsPath,
				fmt.Sprintf(
					`az storage blob download-batch --overwrite true -d %s -s ${CI_COMMIT_SHA}${DRONE_TAG//./-}`,
					step.meta.ArtifactsPath,
				),
				fmt.Sprintf(
					`find %s -type f -exec chmod +x {} \;`,
					step.meta.ArtifactsPath,
				),
			).
				DependsOn(baseStepNames...).
				EnvironmentFromSecret("AZURE_STORAGE_ACCOUNT", "az_storage_account").
				EnvironmentFromSecret("AZURE_STORAGE_USER", "az_storage_user").
				EnvironmentFromSecret("AZURE_STORAGE_PASS", "az_storage_pass").
				EnvironmentFromSecret("AZURE_TENANT", "az_tenant"),
		)

		droneStep := baseDroneStep()

		for k, v := range customPipeline.EnvironmentOverride {
			droneStep.Environment(k, v)
		}

		pipeline.Step(droneStep.DependsOn("load-artifacts"))
	}

	if len(step.Drone.Pipelines) > 0 {
		// add a "save artifacts" step to the default pipeline
		output.Step(
			drone.CustomStep("save-artifacts",
				`az login --service-principal -u "$${AZURE_STORAGE_USER}" -p "$${AZURE_STORAGE_PASS}" --tenant "$${AZURE_TENANT}"`,
				`az storage container create --metadata ci=true -n ${CI_COMMIT_SHA}${DRONE_TAG//./-}`,
				fmt.Sprintf(
					`az storage blob upload-batch --overwrite -s %s -d ${CI_COMMIT_SHA}${DRONE_TAG//./-}`,
					step.meta.ArtifactsPath,
				),
			).
				DependsOn(step.Name()).
				EnvironmentFromSecret("AZURE_STORAGE_ACCOUNT", "az_storage_account").
				EnvironmentFromSecret("AZURE_STORAGE_USER", "az_storage_user").
				EnvironmentFromSecret("AZURE_STORAGE_PASS", "az_storage_pass").
				EnvironmentFromSecret("AZURE_TENANT", "az_tenant"),
		)
	}

	return nil
}

// DroneEnabled implements drone.CustomCompiler.
func (step *Step) DroneEnabled() bool {
	return step.Drone.Enabled
}

// CompileGitHubWorkflow implements ghworkflow.Compiler.
//
//nolint:gocognit,gocyclo,cyclop,maintidx
func (step *Step) CompileGitHubWorkflow(output *ghworkflow.Output) error {
	if !step.GHAction.Enabled {
		return nil
	}

	workflowStep := ghworkflow.Step(step.Name()).SetMakeStep(step.Name())
	if step.SudoInCI {
		workflowStep.SetSudo()
	}

	for k, v := range step.GHAction.Environment {
		workflowStep.SetEnv(k, v)
	}

	if err := workflowStep.SetConditions(step.GHAction.Condition); err != nil {
		return err
	}

	steps := []*ghworkflow.JobStep{workflowStep}

	if len(step.GHAction.Jobs) > 0 {
		steps = append(
			steps,
			ghworkflow.Step("Retrieve PR labels").
				SetID("retrieve-pr-labels").
				SetUses("actions/github-script@"+config.GitHubScriptActionVersion).
				SetWith("retries", "3").
				SetWith("script", strings.TrimPrefix(ghworkflow.IssueLabelRetrieveScript, "\n")),
		)

		output.AddOutputs("default", map[string]string{
			"labels": "${{ steps.retrieve-pr-labels.outputs.result }}",
		})
	}

	additionalArtifactsSteps := []*ghworkflow.JobStep{}

	if step.GHAction.Artifacts.Enabled {
		saveArtifactsStep := ghworkflow.Step("save-artifacts").
			SetUses("actions/upload-artifact@"+config.UploadArtifactActionVersion).
			SetWith("name", "artifacts").
			SetWith("path", step.meta.ArtifactsPath+"\n"+strings.Join(step.GHAction.Artifacts.ExtraPaths, "\n")).
			SetWith("retention-days", "5")

		if retentionDays := step.GHAction.Artifacts.RetentionDays; retentionDays != "" {
			saveArtifactsStep.SetWith("retention-days", retentionDays)
		}

		if step.GHAction.Artifacts.ContinueOnError {
			saveArtifactsStep.SetContinueOnError()
		}

		steps = append(
			steps,
			ghworkflow.Step("Generate executable list").
				SetCommand(fmt.Sprintf("find %s -type f -executable > %s/executable-artifacts", step.meta.ArtifactsPath, step.meta.ArtifactsPath)),
			saveArtifactsStep,
		)

		for _, additionalArtifact := range step.GHAction.Artifacts.Additional {
			artifactStep := ghworkflow.Step(fmt.Sprintf("save-%s-artifacts", additionalArtifact.Name)).
				SetUses("actions/upload-artifact@"+config.UploadArtifactActionVersion).
				SetWith("name", additionalArtifact.Name).
				SetWith("path", strings.Join(additionalArtifact.Paths, "\n")).
				SetWith("retention-days", "5")

			if retentionDays := additionalArtifact.RetentionDays; retentionDays != "" {
				artifactStep.SetWith("retention-days", retentionDays)
			}

			if additionalArtifact.Always {
				if err := artifactStep.SetConditions("always"); err != nil {
					return err
				}
			}

			additionalArtifactsSteps = append(additionalArtifactsSteps, artifactStep)
		}
	}

	steps = append(steps, additionalArtifactsSteps...)

	output.AddStep(
		"default",
		steps...,
	)

	runnerLabels := []string{
		ghworkflow.HostedRunner,
	}

	for _, job := range step.GHAction.Jobs {
		conditions := xslices.Map(job.TriggerLabels, func(label string) string {
			return fmt.Sprintf("contains(fromJSON(needs.default.outputs.labels), '%s')", label)
		})

		var artifactSteps []*ghworkflow.JobStep

		if step.GHAction.Artifacts.Enabled {
			for _, additionalArtifact := range step.GHAction.Artifacts.Additional {
				artifactStep := ghworkflow.Step(fmt.Sprintf("save-%s-artifacts", additionalArtifact.Name)).
					SetUses("actions/upload-artifact@"+config.UploadArtifactActionVersion).
					SetWith("name", additionalArtifact.Name+"-"+job.Name).
					SetWith("path", strings.Join(additionalArtifact.Paths, "\n")).
					SetWith("retention-days", "5")

				if additionalArtifact.Always {
					if err := artifactStep.SetConditions("always"); err != nil {
						return err
					}
				}

				if additionalArtifact.ContinueOnError {
					artifactStep.SetContinueOnError()
				}

				artifactSteps = append(artifactSteps, artifactStep)
			}
		}

		defaultSteps := ghworkflow.DefaultSteps()

		if step.GHAction.SOPS {
			defaultSteps = append(
				defaultSteps,
				ghworkflow.SOPSSteps()...,
			)
		}

		output.AddJob(job.Name, &ghworkflow.Job{
			RunsOn: append(runnerLabels, job.RunnerLabels...),
			If:     strings.Join(conditions, " || "),
			Needs:  []string{"default"},
			Steps:  defaultSteps,
		})

		var steps []*ghworkflow.JobStep

		if step.GHAction.Artifacts.Enabled {
			steps = append(
				steps,
				ghworkflow.Step("Download artifacts").
					SetUses("actions/download-artifact@"+config.DownloadArtifactActionVersion).
					SetWith("name", "artifacts").
					SetWith("path", step.meta.ArtifactsPath),
				ghworkflow.Step("Fix artifact permissions").
					SetCommand(fmt.Sprintf("xargs -a %s/executable-artifacts -I {} chmod +x {}", step.meta.ArtifactsPath)),
			)
		}

		workflowStep := ghworkflow.Step(step.Name()).SetMakeStep(step.Name())

		if step.SudoInCI {
			workflowStep.SetSudo()
		}

		for k, v := range step.GHAction.Environment {
			workflowStep.SetEnv(k, v)
		}

		for k, v := range job.EnvironmentOverride {
			workflowStep.SetEnv(k, v)
		}

		steps = append(steps, workflowStep)
		steps = append(steps, artifactSteps...)

		output.AddStep(
			job.Name,
			steps...,
		)

		if len(job.Crons) > 0 {
			workflowName := job.Name + "-cron"

			output.AddSlackNotify(workflowName)

			steps := []*ghworkflow.JobStep{workflowStep}
			steps = append(steps, additionalArtifactsSteps...)

			defaultSteps := ghworkflow.DefaultSteps()

			if step.GHAction.SOPS {
				defaultSteps = append(
					defaultSteps,
					ghworkflow.SOPSSteps()...,
				)
			}

			output.AddWorkflow(
				workflowName,
				&ghworkflow.Workflow{
					Name: workflowName,
					// https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#example-using-a-fallback-value
					Concurrency: ghworkflow.Concurrency{
						Group:            "${{ github.head_ref || github.run_id }}",
						CancelInProgress: true,
					},
					On: ghworkflow.On{
						Schedule: xslices.Map(job.Crons, func(cron string) ghworkflow.Schedule {
							return ghworkflow.Schedule{
								Cron: cron,
							}
						}),
					},
					Jobs: map[string]*ghworkflow.Job{
						"default": {
							RunsOn: append(runnerLabels, job.RunnerLabels...),
							Steps: append(
								defaultSteps,
								steps...,
							),
						},
					},
				},
			)
		}
	}

	return nil
}

// CompileMakefile implements makefile.Compiler.
func (step *Step) CompileMakefile(output *makefile.Output) error {
	if !step.Makefile.Enabled {
		return nil
	}

	for _, variable := range step.Makefile.Variables {
		output.VariableGroup(makefile.VariableGroupExtra).
			Variable(makefile.OverridableVariable(variable.Name, variable.DefaultValue))
	}

	target := output.Target(step.Name()).
		Depends(step.Makefile.Depends...).
		Script(step.Makefile.Script...)

	if step.Makefile.Phony {
		target.Phony()
	}

	return nil
}

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
			Steps       []struct {
				Script *struct {
					Command string   `yaml:"command"`
					Cache   []string `yaml:"cache"`
				} `yaml:"script"`
				Copy *struct {
					From string `yaml:"from"`
					Src  string `yaml:"src"`
					Dst  string `yaml:"dst"`
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
				Name   string   `yaml:"name"`
				Paths  []string `yaml:"paths"`
				Always bool     `yaml:"always"`
			} `yaml:"additional"`
			Enabled bool `yaml:"enabled"`
		} `yaml:"artifacts"`
		Enabled bool `yaml:"enabled"`
	} `yaml:"ghaction"`
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

		for _, stageStep := range stage.Steps {
			switch {
			case stageStep.Arg != "":
				s.Step(dockerstep.Arg(stageStep.Arg))
			case stageStep.Script != nil:
				script := dockerstep.Script(stageStep.Script.Command)
				for _, cache := range stageStep.Script.Cache {
					script.MountCache(cache)
				}

				s.Step(script)
			case stageStep.Copy != nil:
				copyStep := dockerstep.Copy(stageStep.Copy.Src, stageStep.Copy.Dst)
				if stageStep.Copy.From != "" {
					copyStep.From(stageStep.Copy.From)
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
			baseInput.(baseDroneStepper).BuildBaseDroneSteps(pipeline) //nolint:forcetypeassert // type is checked in GatherMatchingInputsRecursive

			baseStepNames = append(baseStepNames, baseInput.Name())
		}

		pipeline.Step(
			drone.CustomStep("load-artifacts",
				`az login --service-principal -u "$${AZURE_STORAGE_USER}" -p "$${AZURE_STORAGE_PASS}" --tenant "$${AZURE_TENANT}"`,
				fmt.Sprintf(
					`mkdir -p %s`,
					step.meta.ArtifactsPath,
				),
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
func (step *Step) CompileGitHubWorkflow(output *ghworkflow.Output) error {
	if !step.GHAction.Enabled {
		return nil
	}

	workflowStep := ghworkflow.MakeStep(step.Name())

	for k, v := range step.GHAction.Environment {
		workflowStep.SetEnv(k, v)
	}

	steps := []*ghworkflow.Step{
		workflowStep,
		{
			Name: "Retrieve workflow info",
			ID:   "workflow-run-info",
			Uses: fmt.Sprintf("potiuk/get-workflow-origin@%s", config.GetWorkflowOriginActionVersion),
			With: map[string]string{
				"token": "${{ secrets.GITHUB_TOKEN }}",
			},
		},
	}

	output.AddOutputs("default", map[string]string{
		"labels": "${{ steps.workflow-run-info.outputs.pullRequestLabels }}",
	})

	additionalArtifactsSteps := []*ghworkflow.Step{}

	if step.GHAction.Artifacts.Enabled {
		steps = append(steps,
			&ghworkflow.Step{
				Name: "Generate executable list",
				Run:  fmt.Sprintf("find %s -type f -executable > %s/executable-artifacts", step.meta.ArtifactsPath, step.meta.ArtifactsPath) + "\n",
			},
			&ghworkflow.Step{
				Name: "save-artifacts",
				Uses: fmt.Sprintf("actions/upload-artifact@%s", config.UploadArtifactActionVersion),
				With: map[string]string{
					"name":           "artifacts",
					"path":           step.meta.ArtifactsPath + "\n" + strings.Join(step.GHAction.Artifacts.ExtraPaths, "\n"),
					"retention-days": "5",
				},
			},
		)

		for _, additionalArtifact := range step.GHAction.Artifacts.Additional {
			artifactStep := &ghworkflow.Step{
				Name: fmt.Sprintf("save-%s-artifacts", additionalArtifact.Name),
				Uses: fmt.Sprintf("actions/upload-artifact@%s", config.UploadArtifactActionVersion),
				With: map[string]string{
					"name":           additionalArtifact.Name,
					"path":           strings.Join(additionalArtifact.Paths, "\n"),
					"retention-days": "5",
				},
			}

			if additionalArtifact.Always {
				artifactStep.If = "always()"
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
			return fmt.Sprintf("contains(needs.default.outputs.labels, '%s')", label)
		})

		output.AddJob(job.Name, &ghworkflow.Job{
			RunsOn:   append(runnerLabels, job.RunnerLabels...),
			If:       strings.Join(conditions, " || "),
			Needs:    []string{"default"},
			Services: ghworkflow.DefaultServices(),
			Steps:    ghworkflow.DefaultSteps(),
		})

		steps := []*ghworkflow.Step{
			{
				Name: "Download artifacts",
				Uses: fmt.Sprintf("actions/download-artifact@%s", config.DownloadArtifactActionVersion),
				With: map[string]string{
					"name": "artifacts",
					"path": step.meta.ArtifactsPath,
				},
			},
			{
				Name: "Fix artifact permissions",
				Run:  fmt.Sprintf("xargs -a %s/executable-artifacts -I {} chmod +x {}", step.meta.ArtifactsPath) + "\n",
			},
		}

		workflowStep := ghworkflow.MakeStep(step.Name())

		for k, v := range step.GHAction.Environment {
			workflowStep.SetEnv(k, v)
		}

		for k, v := range job.EnvironmentOverride {
			workflowStep.SetEnv(k, v)
		}

		steps = append(steps, workflowStep)

		output.AddStep(
			job.Name,
			steps...,
		)

		if len(job.Crons) > 0 {
			workflowName := fmt.Sprintf("%s-cron", job.Name)

			output.AddSlackNotify(workflowName)

			steps := []*ghworkflow.Step{workflowStep}
			steps = append(steps, additionalArtifactsSteps...)

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
							RunsOn:   append(runnerLabels, job.RunnerLabels...),
							Services: ghworkflow.DefaultServices(),
							Steps: append(
								ghworkflow.DefaultSteps(),
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

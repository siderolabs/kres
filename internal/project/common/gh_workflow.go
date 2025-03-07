// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/siderolabs/gen/xslices"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Job defines options for jobs.
type Job struct {
	Name          string         `yaml:"name"`
	BuildxOptions *BuildxOptions `yaml:"buildxOptions,omitempty"`
	Conditions    []string       `yaml:"conditions,omitempty"`
	Crons         []string       `yaml:"crons,omitempty"`
	Depends       []string       `yaml:"depends,omitempty"`
	Runners       []string       `yaml:"runners,omitempty"`
	TriggerLabels []string       `yaml:"triggerLabels,omitempty"`
	Steps         []Step         `yaml:"steps,omitempty"`
	SOPS          bool           `yaml:"sops"`
}

// BuildxOptions defines options for buildx.
type BuildxOptions struct {
	Enabled      bool `yaml:"enabled"`
	CrossBuilder bool `yaml:"crossBuilder"`
}

// Step defines options for steps.
type Step struct { //nolint:govet
	Name              string             `yaml:"name"`
	Command           string             `yaml:"command"`
	NonMakeStep       bool               `yaml:"nonMakeStep,omitempty"`
	ArtifactStep      *ArtifactStep      `yaml:"artifactStep,omitempty"`
	CheckoutStep      *CheckoutStep      `yaml:"checkoutStep,omitempty"`
	CoverageStep      *CoverageStep      `yaml:"coverageStep,omitempty"`
	TerraformStep     bool               `yaml:"terraformStep,omitempty"`
	RegistryLoginStep *RegistryLoginStep `yaml:"registryLoginStep,omitempty"`
	ReleaseStep       *ReleaseStep       `yaml:"releaseStep,omitempty"`
	WithSudo          bool               `yaml:"withSudo,omitempty"`
	Arguments         []string           `yaml:"arguments,omitempty"`
	Environment       map[string]string  `yaml:"environment"`
	TimeoutMinutes    int                `yaml:"timeoutMinutes,omitempty"`
	ContinueOnError   bool               `yaml:"continueOnError,omitempty"`
	Conditions        []string           `yaml:"conditions,omitempty"`
}

// ArtifactStep defines options for artifact steps.
type ArtifactStep struct {
	Type                            string   `yaml:"type"`
	ArtifactName                    string   `yaml:"artifactName"`
	ArtifactPath                    string   `yaml:"artifactPath"`
	RetentionDays                   string   `yaml:"retentionDays,omitempty"`
	AdditionalArtifacts             []string `yaml:"additionalArtifacts,omitempty"`
	DisableExecutableListGeneration bool     `yaml:"disableExecutableListGeneration"`
}

// CheckoutStep defines options for checkout steps.
type CheckoutStep struct {
	Repository string `yaml:"repository"`
	Ref        string `yaml:"ref"`
	Path       string `yaml:"path"`
}

// CoverageStep defines options for coverage steps.
type CoverageStep struct {
	Files []string `yaml:"files"`
}

// ReleaseStep defines options for release steps.
type ReleaseStep struct {
	BaseDirectory     string   `yaml:"baseDirectory"`
	ReleaseNotes      string   `yaml:"releaseNotes"`
	Artifacts         []string `yaml:"artifacts"`
	GenerateChecksums bool     `yaml:"generateChecksums"`
}

// RegistryLoginStep defines options for registry login steps.
type RegistryLoginStep struct {
	Registry string `yaml:"registry"`
}

// GHWorkflow is a node that represents the GitHub workflow configuration.
type GHWorkflow struct {
	dag.BaseNode

	meta *meta.Options

	CustomRunners []string `yaml:"customRunners,omitempty"`
	Jobs          []Job    `yaml:"jobs"`
}

// NewGHWorkflow creates a new GHWorkflow node.
func NewGHWorkflow(meta *meta.Options) *GHWorkflow {
	return &GHWorkflow{
		BaseNode: dag.NewBaseNode("ghworkflow"),

		meta: meta,
	}
}

// CompileGitHubWorkflow implements ghworkflow.Compiler.
//
//nolint:gocognit,gocyclo,cyclop,maintidx
func (gh *GHWorkflow) CompileGitHubWorkflow(o *ghworkflow.Output) error {
	if !gh.meta.CompileGithubWorkflowsOnly {
		o.SetRunners(gh.CustomRunners...)

		return nil
	}

	touchedJobs := make(map[string]struct{})

	for _, job := range gh.Jobs {
		jobDef := &ghworkflow.Job{
			If:          ghworkflow.DefaultSkipCondition,
			RunsOn:      job.Runners,
			Permissions: ghworkflow.DefaultJobPermissions(),
			Needs:       job.Depends,
			Steps:       ghworkflow.CommonSteps(),
		}

		if err := jobDef.SetConditions(job.Conditions...); err != nil {
			return err
		}

		if job.BuildxOptions != nil && job.BuildxOptions.Enabled {
			jobDef.Steps = ghworkflow.DefaultSteps()

			if job.BuildxOptions.CrossBuilder {
				jobDef.Steps = ghworkflow.DefaultPkgsSteps()
			}
		}

		if job.SOPS {
			jobDef.Steps = append(jobDef.Steps, ghworkflow.SOPSSteps()...)
		}

		for _, step := range job.Steps {
			if step.ArtifactStep != nil {
				var steps []*ghworkflow.JobStep

				switch step.ArtifactStep.Type {
				case "upload":
					saveArtifactsStep := ghworkflow.Step("save artifacts").
						SetUses("actions/upload-artifact@"+config.UploadArtifactActionVersion).
						SetWith("name", step.ArtifactStep.ArtifactName).
						SetWith("path", step.ArtifactStep.ArtifactPath+"\n"+strings.Join(step.ArtifactStep.AdditionalArtifacts, "\n")).
						SetWith("retention-days", "5")

					if retentionDays := step.ArtifactStep.RetentionDays; retentionDays != "" {
						saveArtifactsStep.SetWith("retention-days", retentionDays)
					}

					if step.ContinueOnError {
						saveArtifactsStep.SetContinueOnError()
					}

					if err := saveArtifactsStep.SetConditions(step.Conditions...); err != nil {
						return err
					}

					if !step.ArtifactStep.DisableExecutableListGeneration {
						generateExecutableListStep := ghworkflow.Step("Generate executable list").
							SetCommand(fmt.Sprintf("find %s -type f -executable > %s/executable-artifacts", step.ArtifactStep.ArtifactPath, step.ArtifactStep.ArtifactPath))

						if err := generateExecutableListStep.SetConditions(step.Conditions...); err != nil {
							return err
						}

						steps = append(steps, generateExecutableListStep)
					}

					steps = append(steps, saveArtifactsStep)
				case "download":
					downloadArtifactsStep := ghworkflow.Step("Download artifacts").
						SetUses("actions/download-artifact@"+config.DownloadArtifactActionVersion).
						SetWith("name", step.ArtifactStep.ArtifactName).
						SetWith("path", step.ArtifactStep.ArtifactPath)

					if step.ContinueOnError {
						downloadArtifactsStep.SetContinueOnError()
					}

					if err := downloadArtifactsStep.SetConditions(step.Conditions...); err != nil {
						return err
					}

					steps = append(steps, downloadArtifactsStep)

					if !step.ArtifactStep.DisableExecutableListGeneration {
						fixArtifactPermissionsStep := ghworkflow.Step("Fix artifact permissions").
							SetCommand(fmt.Sprintf("xargs -a %s/executable-artifacts -I {} chmod +x {}", step.ArtifactStep.ArtifactPath))

						if err := fixArtifactPermissionsStep.SetConditions(step.Conditions...); err != nil {
							return err
						}

						steps = append(steps, fixArtifactPermissionsStep)
					}
				default:
					return fmt.Errorf("unknown artifact step type: %s", step.ArtifactStep.Type)
				}

				jobDef.Steps = append(jobDef.Steps, steps...)

				continue
			}

			if step.CheckoutStep != nil {
				checkoutStep := ghworkflow.Step(step.Name).
					SetUses("actions/checkout@"+config.CheckOutActionVersion).
					SetWith("repository", step.CheckoutStep.Repository).
					SetWith("ref", step.CheckoutStep.Ref).
					SetWith("path", step.CheckoutStep.Path)

				jobDef.Steps = append(jobDef.Steps, checkoutStep)

				continue
			}

			if step.CoverageStep != nil {
				coverageStep := ghworkflow.Step(step.Name).
					SetUses("codecov/codecov-action@"+config.CodeCovActionVersion).
					SetWith("files", strings.Join(step.CoverageStep.Files, ",")).
					SetWith("token", "${{ secrets.CODECOV_TOKEN }}").
					SetTimeoutMinutes(step.TimeoutMinutes)

				jobDef.Steps = append(jobDef.Steps, coverageStep)

				continue
			}

			if step.TerraformStep {
				terraformStep := ghworkflow.Step(step.Name).
					SetUses("hashicorp/setup-terraform@"+config.SetupTerraformActionVersion).
					SetWith("terraform_wrapper", "false")

				jobDef.Steps = append(jobDef.Steps, terraformStep)

				continue
			}

			if step.RegistryLoginStep != nil {
				registryLoginStep := ghworkflow.Step(step.Name).
					SetUses("docker/login-action@"+config.LoginActionVersion).
					SetWith("registry", step.RegistryLoginStep.Registry)

				if step.RegistryLoginStep.Registry == "ghcr.io" {
					registryLoginStep.SetWith("username", "${{ github.repository_owner }}")
					registryLoginStep.SetWith("password", "${{ secrets.GITHUB_TOKEN }}")
				}

				if err := registryLoginStep.SetConditions(step.Conditions...); err != nil {
					return err
				}

				jobDef.Steps = append(jobDef.Steps, registryLoginStep)

				continue
			}

			if step.ReleaseStep != nil {
				artifacts := xslices.Map(step.ReleaseStep.Artifacts, func(artifact string) string {
					return filepath.Join(step.ReleaseStep.BaseDirectory, artifact)
				})

				releaseStep := ghworkflow.Step(step.Name).
					SetUses("crazy-max/ghaction-github-release@"+config.ReleaseActionVersion).
					SetWith("body_path", filepath.Join(step.ReleaseStep.BaseDirectory, step.ReleaseStep.ReleaseNotes)).
					SetWith("draft", "true").
					SetWith("files", strings.Join(artifacts, "\n"))

				if step.ReleaseStep.GenerateChecksums {
					checkSumCommands := []string{
						fmt.Sprintf("cd %s", step.ReleaseStep.BaseDirectory),
						fmt.Sprintf("sha256sum %s > %s", strings.Join(step.ReleaseStep.Artifacts, " "), "sha256sum.txt"),
						fmt.Sprintf("sha512sum %s > %s", strings.Join(step.ReleaseStep.Artifacts, " "), "sha512sum.txt"),
					}

					checkSumStep := ghworkflow.Step("Generate Checksums").
						SetCommand(strings.Join(checkSumCommands, "\n"))

					releaseStep.
						SetWith("files", strings.Join(artifacts, "\n")+"\n"+filepath.Join(step.ReleaseStep.BaseDirectory, "sha*.txt"))

					jobDef.Steps = append(jobDef.Steps, checkSumStep)
				}

				jobDef.Steps = append(jobDef.Steps, releaseStep)

				continue
			}

			command := step.Command

			stepDef := ghworkflow.Step(step.Name)

			stepDef.SetTimeoutMinutes(step.TimeoutMinutes)

			if step.ContinueOnError {
				stepDef.SetContinueOnError()
			}

			if err := stepDef.SetConditions(step.Conditions...); err != nil {
				return err
			}

			for k, v := range step.Environment {
				stepDef.SetEnv(k, v)
			}

			if step.NonMakeStep {
				if len(step.Arguments) > 0 {
					command += " " + strings.Join(step.Arguments, "")
				}

				stepDef.SetCommand(command)

				jobDef.Steps = append(jobDef.Steps, stepDef)

				continue
			}

			if command == "" {
				command = step.Name
			}

			stepDef.SetMakeStep(command, step.Arguments...)

			if step.WithSudo {
				stepDef.SetSudo()
			}

			jobDef.Steps = append(jobDef.Steps, stepDef)
		}

		if len(job.TriggerLabels) > 0 {
			if len(job.Depends) < 1 {
				return fmt.Errorf("job %s has triggerLabels but no depends", job.Name)
			}

			for _, dep := range job.Depends {
				if _, ok := touchedJobs[dep]; !ok {
					o.AddStep(dep,
						ghworkflow.Step("Retrieve PR labels").
							SetID("retrieve-pr-labels").
							SetUses("actions/github-script@"+config.GitHubScriptActionVersion).
							SetWith("retries", "3").
							SetWith("script", strings.TrimPrefix(ghworkflow.IssueLabelRetrieveScript, "\n")),
					)

					o.AddOutputs(dep, map[string]string{
						"labels": "${{ steps.retrieve-pr-labels.outputs.result }}",
					})

					touchedJobs[dep] = struct{}{}
				}
			}

			conditions := xslices.Map(job.TriggerLabels, func(label string) string {
				return fmt.Sprintf("contains(fromJSON(needs.default.outputs.labels), '%s')", label)
			})

			for range job.TriggerLabels {
				jobDef.If = strings.Join(conditions, " || ")
			}
		}

		if len(job.Crons) > 0 {
			workflowName := job.Name + "-cron"

			o.AddSlackNotify(workflowName)

			o.AddWorkflow(
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
							RunsOn: job.Runners,
							Steps:  jobDef.Steps,
						},
					},
				},
			)

			o.AddWorkflow(
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
							RunsOn:   job.Runners,
							Services: jobDef.Services,
							Steps:    jobDef.Steps,
						},
					},
				},
			)
		}

		o.AddJob(job.Name, jobDef)
	}

	return nil
}

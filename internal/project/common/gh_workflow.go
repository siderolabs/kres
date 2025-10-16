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
	BuildxOptions *BuildxOptions `yaml:"buildxOptions,omitempty"`
	Name          string         `yaml:"name"`
	RunnerGroup   string         `yaml:"runnerGroup,omitempty"`
	Conditions    []string       `yaml:"conditions,omitempty"`
	Crons         []string       `yaml:"crons,omitempty"`
	Depends       []string       `yaml:"depends,omitempty"`
	TriggerLabels []string       `yaml:"triggerLabels,omitempty"`
	Steps         []Step         `yaml:"steps,omitempty"`
	Inputs        []string       `yaml:"inputs,omitempty"`
	Dispatchable  bool           `yaml:"dispatchable"`
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
	BaseDirectory      string   `yaml:"baseDirectory"`
	ReleaseNotes       string   `yaml:"releaseNotes"`
	Artifacts          []string `yaml:"artifacts"`
	GenerateChecksums  bool     `yaml:"generateChecksums"`
	GenerateSignatures bool     `yaml:"generateSignatures"`
}

// RegistryLoginStep defines options for registry login steps.
type RegistryLoginStep struct {
	Registry string `yaml:"registry"`
}

// GHWorkflow is a node that represents the GitHub workflow configuration.
type GHWorkflow struct {
	meta *meta.Options
	dag.BaseNode

	*WorkflowOptions             `yaml:"workflowOptions,omitempty"`
	CIFailuresSlackNotifyChannel string `yaml:"ciFailuresSlackNotifyChannel,omitempty"`
	CustomRunnerGroup            string `yaml:"customRunnerGroup,omitempty"`

	Jobs []Job `yaml:"jobs"`
}

// WorkflowOptions defines options for the workflow.
type WorkflowOptions struct {
	ghworkflow.On `yaml:"on"`
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
		o.SetRunnerGroup(gh.CustomRunnerGroup)

		return nil
	}

	touchedJobs := make(map[string]struct{})

	if gh.WorkflowOptions != nil {
		o.SetWorkflowOn(gh.On)
	}

	for _, job := range gh.Jobs {
		jobDef := &ghworkflow.Job{
			If:          ghworkflow.DefaultSkipCondition,
			RunsOn:      ghworkflow.NewRunsOnGroupLabel(job.RunnerGroup, ""),
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
					SetUses("softprops/action-gh-release@"+config.ReleaseActionVersion).
					SetWith("body_path", filepath.Join(step.ReleaseStep.BaseDirectory, step.ReleaseStep.ReleaseNotes)).
					SetWith("draft", "true").
					SetWith("files", strings.Join(artifacts, "\n"))

				if step.ReleaseStep.GenerateSignatures {
					jobDef.Permissions["id-token"] = "write"

					cosignStep := ghworkflow.Step("Install Cosign").
						SetUses("sigstore/cosign-installer@" + config.CosignInstallActionVerson)

					jobDef.Steps = append(jobDef.Steps, cosignStep)

					signCommands := xslices.Map(artifacts, func(artifact string) string {
						return fmt.Sprintf("cosign sign-blob --bundle %s.bundle --yes %s", artifact, artifact)
					})

					signStep := ghworkflow.Step("Sign artifacts").
						SetCommand(strings.Join(signCommands, "\n"))

					jobDef.Steps = append(jobDef.Steps, signStep)

					releaseStep.SetWith("files", strings.Join(artifacts, "\n")+"\n"+filepath.Join(step.ReleaseStep.BaseDirectory, "*.bundle"))
				}

				if step.ReleaseStep.GenerateChecksums {
					jobDef.Permissions["id-token"] = "write"

					checkSumCommands := []string{
						fmt.Sprintf("cd %s", step.ReleaseStep.BaseDirectory),
						fmt.Sprintf("sha256sum %s > %s", strings.Join(step.ReleaseStep.Artifacts, " "), "sha256sum.txt"),
						fmt.Sprintf("sha512sum %s > %s", strings.Join(step.ReleaseStep.Artifacts, " "), "sha512sum.txt"),
					}

					checkSumStep := ghworkflow.Step("Generate Checksums").
						SetCommand(strings.Join(checkSumCommands, "\n"))

					jobDef.Steps = append(jobDef.Steps, checkSumStep)

					releaseStep.
						SetWith("files", strings.Join(artifacts, "\n")+"\n"+filepath.Join(step.ReleaseStep.BaseDirectory, "sha*.txt"))

					if step.ReleaseStep.GenerateSignatures {
						checkSumSignCommands := []string{
							fmt.Sprintf("cd %s", step.ReleaseStep.BaseDirectory),
							"cosign sign-blob --bundle sha256sum.txt.bundle --yes sha256sum.txt",
							"cosign sign-blob --bundle sha512sum.txt.bundle --yes sha512sum.txt",
						}

						signStep := ghworkflow.Step("Sign checksums").
							SetCommand(strings.Join(checkSumSignCommands, "\n"))

						jobDef.Steps = append(jobDef.Steps, signStep)

						releaseStep.SetWith("files",
							strings.Join(artifacts, "\n")+
								"\n"+
								filepath.Join(step.ReleaseStep.BaseDirectory, "sha*.txt")+"\n"+
								filepath.Join(step.ReleaseStep.BaseDirectory, "*.bundle"),
						)
					}
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
			o.AddSlackNotifyForFailure(workflowName)

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
						ghworkflow.DefaultJobName: {
							RunsOn:   ghworkflow.NewRunsOnGroupLabel(job.RunnerGroup, ""),
							Services: jobDef.Services,
							Steps:    jobDef.Steps,
						},
					},
				},
			)
		}

		o.AddJob(job.Name, job.Dispatchable, jobDef, job.Inputs)
	}

	return nil
}

// DefaultCIFailureSlackNotifyChannel is the default channel for CI failure Slack notifications.
const DefaultCIFailureSlackNotifyChannel = "ci-failure"

// AfterLoad maps back ci failure slack notify channel override or default value to meta.
func (gh *GHWorkflow) AfterLoad() error {
	gh.meta.CIFailureSlackNotifyChannel = gh.CIFailuresSlackNotifyChannel
	if gh.meta.CIFailureSlackNotifyChannel == "" {
		gh.meta.CIFailureSlackNotifyChannel = DefaultCIFailureSlackNotifyChannel
	}

	return nil
}

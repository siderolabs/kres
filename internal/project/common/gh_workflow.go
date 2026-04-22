// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"fmt"
	"maps"
	"path/filepath"
	"regexp"
	"slices"
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
	Matrix        *Matrix        `yaml:"matrix,omitempty"`
	OnWorkflowRun *OnWorkflowRun `yaml:"onWorkflowRun,omitempty"`
	Name          string         `yaml:"name"`
	RunnerGroup   string         `yaml:"runnerGroup,omitempty"`
	TriggerLabels []string       `yaml:"triggerLabels,omitempty"`
	Depends       []string       `yaml:"depends,omitempty"`
	Steps         []Step         `yaml:"steps,omitempty"`
	Inputs        []string       `yaml:"inputs,omitempty"`
	Crons         []string       `yaml:"crons,omitempty"`
	Conditions    []string       `yaml:"conditions,omitempty"`
	Dispatchable  bool           `yaml:"dispatchable"`
	SOPS          bool           `yaml:"sops"`
	CronOnly      bool           `yaml:"cronOnly"`
}

// MatrixEntry is one row of key-value pairs for a matrix include entry.
// Values are interpolated into step commands, env vars, and artifact names
// using ${{ matrix.<key> }} syntax.
type MatrixEntry map[string]string

// MatrixInclude is a matrix include entry combining key-value pairs with
// optional per-entry trigger labels.
type MatrixInclude struct {
	// Values holds the key-value pairs for this matrix entry (all YAML keys
	// that are not explicitly declared as struct fields).
	Values MatrixEntry `yaml:",inline"`
	// TriggerLabels lists labels that trigger only this specific flat job in
	// ci.yaml (in addition to the job-level TriggerLabels which fire all entries).
	TriggerLabels []string `yaml:"triggerLabels,omitempty"`
}

// Matrix configures a GitHub Actions matrix strategy on a triggered workflow,
// and controls how ci.yaml expands the matrix into individual jobs.
type Matrix struct {
	Include       []MatrixInclude `yaml:"include"`
	LabelKeys     []string        `yaml:"labelKeys,omitempty"`
	MaxParallel   int             `yaml:"maxParallel"`
	FlatJobMatrix bool            `yaml:"flatJobMatrix,omitempty"`
}

// OnWorkflowRun defines options for workflow_run triggers.
type OnWorkflowRun struct {
	Workflows []string `yaml:"workflows"`
	Types     []string `yaml:"types,omitempty"`
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
	RunID                           string   `yaml:"runID,omitempty"`
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

	// LabelDescriptions maps PR trigger label names to human-readable
	// descriptions. Used by repository.enableLabels to set descriptions when
	// creating/updating labels.
	LabelDescriptions map[string]string `yaml:"labelDescriptions,omitempty"`

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

// CollectEnforceContexts returns the list of GitHub Actions check names for
// all jobs defined in this workflow, including per-entry names for flat-expanded
// and flatJobMatrix matrix jobs. Jobs that don't run on pull requests (cronOnly
// or gated by conditions that exclude PRs) are skipped. The result is suitable
// for use as required status checks in branch protection.
func (gh *GHWorkflow) CollectEnforceContexts() []string {
	var contexts []string

	for _, job := range gh.Jobs {
		// Skip jobs that can never produce a PR status check:
		//   - cronOnly: scheduled-only, not in ci.yaml
		//   - Dispatchable: workflow_dispatch only (dispatch.yaml, not ci.yaml)
		//   - Conditions that explicitly exclude PRs (only-on-*, except-pull-request)
		if job.CronOnly || job.Dispatchable {
			continue
		}

		// Conditions default to running on PRs when unset; explicit conditions
		// must include "on-pull-request" to qualify as a PR status check.
		if len(job.Conditions) > 0 && !slices.Contains(job.Conditions, "on-pull-request") {
			continue
		}

		if job.Matrix == nil || len(job.Matrix.LabelKeys) == 0 {
			contexts = append(contexts, job.Name)

			continue
		}

		for _, entry := range job.Matrix.Include {
			parts := make([]string, 0, len(job.Matrix.LabelKeys))

			for _, k := range job.Matrix.LabelKeys {
				if v := entry.Values[k]; v != "" {
					parts = append(parts, v)
				}
			}

			suffix := strings.Join(parts, "-")

			if job.Matrix.FlatJobMatrix {
				contexts = append(contexts, suffix)
			} else {
				contexts = append(contexts, job.Name+"-"+suffix)
			}
		}
	}

	return contexts
}

// CollectTriggerLabels returns the deduplicated set of PR labels referenced as
// job-level TriggerLabels or per-entry matrix TriggerLabels, mapped to their
// description (taken from GHWorkflow.LabelDescriptions; empty string if none).
// The result is suitable for provisioning the corresponding labels on the
// GitHub repository.
func (gh *GHWorkflow) CollectTriggerLabels() map[string]string {
	labels := map[string]string{}

	for _, job := range gh.Jobs {
		for _, l := range job.TriggerLabels {
			labels[l] = gh.LabelDescriptions[l]
		}

		if job.Matrix == nil {
			continue
		}

		for _, entry := range job.Matrix.Include {
			for _, l := range entry.TriggerLabels {
				labels[l] = gh.LabelDescriptions[l]
			}
		}
	}

	return labels
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

		// Capture base steps (checkout/buildx/sops) before user steps are appended.
		// compileFlatJobSteps uses these as its starting point.
		baseSteps := jobDef.Steps

		for _, step := range job.Steps {
			if step.ArtifactStep != nil {
				var steps []*ghworkflow.JobStep

				switch step.ArtifactStep.Type {
				case "upload":
					saveArtifactsStep := ghworkflow.Step("save artifacts").
						SetUsesWithComment(
							"actions/upload-artifact@"+config.UploadArtifactActionRef,
							"version: "+config.UploadArtifactActionVersion,
						).
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
						SetUsesWithComment(
							"actions/download-artifact@"+config.DownloadArtifactActionRef,
							"version: "+config.DownloadArtifactActionVersion,
						).
						SetWith("name", step.ArtifactStep.ArtifactName).
						SetWith("path", step.ArtifactStep.ArtifactPath)

					if step.ArtifactStep.RunID != "" {
						downloadArtifactsStep.SetWith("run-id", step.ArtifactStep.RunID).
							SetWith("github-token", "${{ secrets.GITHUB_TOKEN }}")
					}

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
					SetUsesWithComment(
						"actions/checkout@"+config.CheckOutActionRef,
						"version: "+config.CheckOutActionVersion,
					).
					SetWith("repository", step.CheckoutStep.Repository).
					SetWith("ref", step.CheckoutStep.Ref).
					SetWith("path", step.CheckoutStep.Path)

				jobDef.Steps = append(jobDef.Steps, checkoutStep)

				continue
			}

			if step.CoverageStep != nil {
				coverageStep := ghworkflow.Step(step.Name).
					SetUsesWithComment(
						"codecov/codecov-action@"+config.CodeCovActionRef,
						"version: "+config.CodeCovActionVersion,
					).
					SetWith("files", strings.Join(step.CoverageStep.Files, ",")).
					SetWith("token", "${{ secrets.CODECOV_TOKEN }}").
					SetTimeoutMinutes(step.TimeoutMinutes)

				jobDef.Steps = append(jobDef.Steps, coverageStep)

				continue
			}

			if step.TerraformStep {
				terraformStep := ghworkflow.Step(step.Name).
					SetUsesWithComment(
						"hashicorp/setup-terraform@"+config.SetupTerraformActionRef,
						"version: "+config.SetupTerraformActionVersion,
					).
					SetWith("terraform_wrapper", "false")

				jobDef.Steps = append(jobDef.Steps, terraformStep)

				continue
			}

			if step.RegistryLoginStep != nil {
				registryLoginStep := ghworkflow.Step(step.Name).
					SetUsesWithComment(
						"docker/login-action@"+config.LoginActionRef,
						"version: "+config.LoginActionVersion,
					).
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
					SetUsesWithComment(
						"softprops/action-gh-release@"+config.ReleaseActionRef,
						"version: "+config.ReleaseActionVersion,
					).
					SetWith("body_path", filepath.Join(step.ReleaseStep.BaseDirectory, step.ReleaseStep.ReleaseNotes)).
					SetWith("draft", "true").
					SetWith("files", strings.Join(artifacts, "\n"))

				if step.ReleaseStep.GenerateSignatures {
					jobDef.Permissions["id-token"] = "write"

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
				if v != "" {
					stepDef.SetEnv(k, v)
				}
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

		flatJobsAdded := false

		if len(job.TriggerLabels) > 0 {
			if len(job.Depends) < 1 {
				return fmt.Errorf("job %s has triggerLabels but no depends", job.Name)
			}

			// The if: condition reads from needs.default.outputs.labels, so only
			// the "default" job (if it's a dep) needs the retrieve-pr-labels step.
			const labelProvider = ghworkflow.DefaultJobName

			if slices.Contains(job.Depends, labelProvider) {
				if _, ok := touchedJobs[labelProvider]; !ok {
					o.AddStep(labelProvider,
						ghworkflow.Step("Retrieve PR labels").
							SetID("retrieve-pr-labels").
							SetUsesWithComment(
								"actions/github-script@"+config.GitHubScriptActionRef,
								"version: "+config.GitHubScriptActionVersion,
							).
							SetWith("retries", "3").
							SetWith("script", strings.TrimPrefix(ghworkflow.IssueLabelRetrieveScript, "\n")),
					)

					o.AddOutputs(labelProvider, map[string]string{
						"labels": "${{ steps.retrieve-pr-labels.outputs.result }}",
					})

					touchedJobs[labelProvider] = struct{}{}
				}
			}

			if job.Matrix != nil && len(job.Matrix.LabelKeys) > 0 && !job.Matrix.FlatJobMatrix {
				// Flat job expansion: emit one individual ci.yaml job per matrix entry.
				// Job-level triggerLabels fire all flat jobs; per-entry TriggerLabels
				// fire only that specific entry's flat job.
				coarseConditions := xslices.Map(job.TriggerLabels, func(label string) string {
					return fmt.Sprintf("contains(fromJSON(needs.default.outputs.labels), '%s')", label)
				})

				for _, entry := range job.Matrix.Include {
					suffixParts := make([]string, 0, len(job.Matrix.LabelKeys))

					for _, k := range job.Matrix.LabelKeys {
						if v := entry.Values[k]; v != "" {
							suffixParts = append(suffixParts, v)
						}
					}

					suffix := strings.Join(suffixParts, "-")
					flatJobName := job.Name + "-" + suffix
					entryLabel := "integration/" + strings.TrimPrefix(flatJobName, "integration-")
					entryCondition := fmt.Sprintf("contains(fromJSON(needs.default.outputs.labels), '%s')", entryLabel)

					allConditions := append([]string{entryCondition}, coarseConditions...)

					for _, label := range entry.TriggerLabels {
						allConditions = append(allConditions, fmt.Sprintf("contains(fromJSON(needs.default.outputs.labels), '%s')", label))
					}

					flatSteps, err := compileFlatJobSteps(job, entry.Values, baseSteps)
					if err != nil {
						return err
					}

					flatJob := &ghworkflow.Job{
						If:          strings.Join(allConditions, " || "),
						RunsOn:      ghworkflow.NewRunsOnGroupLabel(job.RunnerGroup, ""),
						Permissions: ghworkflow.DefaultJobPermissions(),
						Needs:       job.Depends,
						Steps:       flatSteps,
					}

					o.AddJob(flatJobName, job.Dispatchable, flatJob, job.Inputs)
				}

				flatJobsAdded = true
			} else {
				conditions := xslices.Map(job.TriggerLabels, func(label string) string {
					return fmt.Sprintf("contains(fromJSON(needs.default.outputs.labels), '%s')", label)
				})

				jobDef.If = strings.Join(conditions, " || ")

				if job.Matrix != nil && job.Matrix.FlatJobMatrix {
					failFast := false

					jobDef.Strategy = &ghworkflow.Strategy{
						MaxParallel: job.Matrix.MaxParallel,
						FailFast:    &failFast,
						Matrix: &ghworkflow.StrategyMatrix{
							Include: xslices.Map(job.Matrix.Include, func(e MatrixInclude) map[string]string {
								result := make(map[string]string, len(e.Values))
								for k, v := range e.Values {
									if v != "" {
										result[k] = v
									}
								}

								return result
							}),
						},
					}

					if len(job.Matrix.LabelKeys) > 0 {
						parts := make([]string, 0, len(job.Matrix.LabelKeys))
						for _, k := range job.Matrix.LabelKeys {
							parts = append(parts, "${{ matrix."+k+" }}")
						}

						jobDef.Name = strings.Join(parts, "-")
					}
				}
			}
		}

		if job.OnWorkflowRun != nil {
			if len(job.OnWorkflowRun.Workflows) == 0 {
				return fmt.Errorf("job %s has onWorkflowRun set but no workflows specified", job.Name)
			}

			workflowName := job.Name + "-triggered"

			o.AddSlackNotify(workflowName)
			o.AddSlackNotifyForFailure(workflowName)

			triggeredJob := &ghworkflow.Job{
				If:     "github.event.workflow_run.conclusion == 'success'",
				RunsOn: ghworkflow.NewRunsOnGroupLabel(job.RunnerGroup, ""),
				Permissions: map[string]string{
					"actions": "read",
				},
				Services: jobDef.Services,
				Steps:    injectTriggeredRunID(jobDef.Steps),
			}

			if job.Matrix != nil {
				failFast := false

				triggeredJob.Strategy = &ghworkflow.Strategy{
					MaxParallel: job.Matrix.MaxParallel,
					FailFast:    &failFast,
					Matrix: &ghworkflow.StrategyMatrix{
						Include: xslices.Map(job.Matrix.Include, func(e MatrixInclude) map[string]string {
							result := make(map[string]string, len(e.Values))
							for k, v := range e.Values {
								if v != "" {
									result[k] = v
								}
							}

							return result
						}),
					},
				}

				if len(job.Matrix.LabelKeys) > 0 {
					parts := make([]string, 0, len(job.Matrix.LabelKeys))
					for _, k := range job.Matrix.LabelKeys {
						parts = append(parts, "${{ matrix."+k+" }}")
					}

					triggeredJob.Name = strings.Join(parts, "-")
				} else if len(job.Matrix.Include) > 0 {
					for k := range job.Matrix.Include[0].Values {
						triggeredJob.Name = "${{ matrix." + k + " }}"

						break
					}
				}
			}

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
						WorkFlowRun: ghworkflow.WorkFlowRun{
							Workflows: job.OnWorkflowRun.Workflows,
							Types:     job.OnWorkflowRun.Types,
						},
					},
					Jobs: map[string]*ghworkflow.Job{
						ghworkflow.DefaultJobName: triggeredJob,
					},
				},
			)
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

		if !job.CronOnly && !flatJobsAdded {
			o.AddJob(job.Name, job.Dispatchable, jobDef, job.Inputs)
		}
	}

	return nil
}

// matrixSubst replaces every ${{ matrix.<key> }} placeholder in s with the
// corresponding value from entry.  Placeholders for keys not present in entry
// are left unchanged.
func matrixSubst(s string, entry MatrixEntry) string {
	for k, v := range entry {
		s = strings.ReplaceAll(s, "${{ matrix."+k+" }}", v)
	}

	return s
}

// matrixPlaceholder matches any remaining ${{ matrix.<key> }} expression.
var matrixPlaceholder = regexp.MustCompile(`\$\{\{\s*matrix\.\w+\s*\}\}`)

// matrixSubstFull is like matrixSubst but additionally replaces any
// ${{ matrix.<key> }} placeholders that were not resolved (i.e. absent from
// the entry) with an empty string. Use this for flat ci.yaml job generation
// where all matrix expressions must be resolved at compile time.
func matrixSubstFull(s string, entry MatrixEntry) string {
	return matrixPlaceholder.ReplaceAllString(matrixSubst(s, entry), "")
}

// resolveConditionsForEntry pre-processes step conditions for a specific matrix
// entry (flat ci.yaml job compilation).
//
// For a condition of the form "matrix.<key>":
//   - entry[key] is non-empty  → condition satisfied; no if-guard is emitted
//   - entry[key] is empty/missing → the step must be dropped; returns (nil, false)
//
// All other conditions are passed through unchanged in the returned slice.
func resolveConditionsForEntry(conditions []string, entry MatrixEntry) ([]string, bool) {
	var resolved []string

	for _, cond := range conditions {
		if key, ok := strings.CutPrefix(cond, "matrix."); ok {
			if entry[key] == "" {
				return nil, false
			}
			// condition satisfied; no guard needed — don't append
		} else {
			resolved = append(resolved, cond)
		}
	}

	return resolved, true
}

// compileFlatJobSteps compiles job.Steps for a specific matrix entry, producing
// a concrete step list for a flat ci.yaml job.
//
//   - Matrix conditions are resolved at compile time (steps are dropped when the
//     entry does not satisfy a matrix.<key> condition).
//   - ${{ matrix.<key> }} expressions in commands, environment values, artifact
//     names, and runID are substituted with entry[key].
//
// baseSteps (CommonSteps / DefaultSteps) are prepended unchanged.
//
//nolint:cyclop,gocognit,cyclop,gocyclo,maintidx
func compileFlatJobSteps(job Job, entry MatrixEntry, baseSteps []*ghworkflow.JobStep) ([]*ghworkflow.JobStep, error) {
	result := baseSteps

	for _, step := range job.Steps {
		resolvedConditions, include := resolveConditionsForEntry(step.Conditions, entry)
		if !include {
			continue
		}

		if step.ArtifactStep != nil {
			var steps []*ghworkflow.JobStep

			switch step.ArtifactStep.Type {
			case "upload":
				artifactName := matrixSubstFull(step.ArtifactStep.ArtifactName, entry)
				artifactPath := step.ArtifactStep.ArtifactPath

				saveStep := ghworkflow.Step("save artifacts").
					SetUsesWithComment(
						"actions/upload-artifact@"+config.UploadArtifactActionRef,
						"version: "+config.UploadArtifactActionVersion,
					).
					SetWith("name", artifactName).
					SetWith("path", artifactPath+"\n"+strings.Join(step.ArtifactStep.AdditionalArtifacts, "\n")).
					SetWith("retention-days", "5")

				if step.ArtifactStep.RetentionDays != "" {
					saveStep.SetWith("retention-days", step.ArtifactStep.RetentionDays)
				}

				if step.ContinueOnError {
					saveStep.SetContinueOnError()
				}

				if err := saveStep.SetConditions(resolvedConditions...); err != nil {
					return nil, err
				}

				if !step.ArtifactStep.DisableExecutableListGeneration {
					genStep := ghworkflow.Step("Generate executable list").
						SetCommand(fmt.Sprintf("find %s -type f -executable > %s/executable-artifacts", artifactPath, artifactPath))

					if err := genStep.SetConditions(resolvedConditions...); err != nil {
						return nil, err
					}

					steps = append(steps, genStep)
				}

				steps = append(steps, saveStep)

			case "download":
				artifactName := matrixSubstFull(step.ArtifactStep.ArtifactName, entry)

				dlStep := ghworkflow.Step("Download artifacts").
					SetUsesWithComment(
						"actions/download-artifact@"+config.DownloadArtifactActionRef,
						"version: "+config.DownloadArtifactActionVersion,
					).
					SetWith("name", artifactName).
					SetWith("path", step.ArtifactStep.ArtifactPath)

				if step.ArtifactStep.RunID != "" {
					dlStep.SetWith("run-id", matrixSubstFull(step.ArtifactStep.RunID, entry)).
						SetWith("github-token", "${{ secrets.GITHUB_TOKEN }}")
				}

				if step.ContinueOnError {
					dlStep.SetContinueOnError()
				}

				if err := dlStep.SetConditions(resolvedConditions...); err != nil {
					return nil, err
				}

				steps = append(steps, dlStep)

				if !step.ArtifactStep.DisableExecutableListGeneration {
					fixStep := ghworkflow.Step("Fix artifact permissions").
						SetCommand(fmt.Sprintf("xargs -a %s/executable-artifacts -I {} chmod +x {}", step.ArtifactStep.ArtifactPath))

					if err := fixStep.SetConditions(resolvedConditions...); err != nil {
						return nil, err
					}

					steps = append(steps, fixStep)
				}

			default:
				return nil, fmt.Errorf("unknown artifact step type: %s", step.ArtifactStep.Type)
			}

			result = append(result, steps...)

			continue
		}

		if step.CheckoutStep != nil {
			result = append(result, ghworkflow.Step(step.Name).
				SetUsesWithComment(
					"actions/checkout@"+config.CheckOutActionRef,
					"version: "+config.CheckOutActionVersion,
				).
				SetWith("repository", step.CheckoutStep.Repository).
				SetWith("ref", step.CheckoutStep.Ref).
				SetWith("path", step.CheckoutStep.Path),
			)

			continue
		}

		if step.CoverageStep != nil {
			result = append(result, ghworkflow.Step(step.Name).
				SetUsesWithComment(
					"codecov/codecov-action@"+config.CodeCovActionRef,
					"version: "+config.CodeCovActionVersion,
				).
				SetWith("files", strings.Join(step.CoverageStep.Files, ",")).
				SetWith("token", "${{ secrets.CODECOV_TOKEN }}").
				SetTimeoutMinutes(step.TimeoutMinutes),
			)

			continue
		}

		if step.TerraformStep {
			result = append(result, ghworkflow.Step(step.Name).
				SetUsesWithComment(
					"hashicorp/setup-terraform@"+config.SetupTerraformActionRef,
					"version: "+config.SetupTerraformActionVersion,
				).
				SetWith("terraform_wrapper", "false"),
			)

			continue
		}

		if step.RegistryLoginStep != nil {
			loginStep := ghworkflow.Step(step.Name).
				SetUsesWithComment(
					"docker/login-action@"+config.LoginActionRef,
					"version: "+config.LoginActionVersion,
				).
				SetWith("registry", step.RegistryLoginStep.Registry)

			if step.RegistryLoginStep.Registry == "ghcr.io" {
				loginStep.SetWith("username", "${{ github.repository_owner }}")
				loginStep.SetWith("password", "${{ secrets.GITHUB_TOKEN }}")
			}

			if err := loginStep.SetConditions(resolvedConditions...); err != nil {
				return nil, err
			}

			result = append(result, loginStep)

			continue
		}

		// Plain command step
		command := matrixSubstFull(step.Command, entry)
		if command == "" {
			command = step.Name
		}

		stepDef := ghworkflow.Step(step.Name)
		stepDef.SetTimeoutMinutes(step.TimeoutMinutes)

		if step.ContinueOnError {
			stepDef.SetContinueOnError()
		}

		if err := stepDef.SetConditions(resolvedConditions...); err != nil {
			return nil, err
		}

		for k, v := range step.Environment {
			if val := matrixSubstFull(v, entry); val != "" {
				stepDef.SetEnv(k, val)
			}
		}

		if step.NonMakeStep {
			if len(step.Arguments) > 0 {
				command += " " + strings.Join(step.Arguments, "")
			}

			stepDef.SetCommand(command)
			result = append(result, stepDef)

			continue
		}

		stepDef.SetMakeStep(command, step.Arguments...)

		if step.WithSudo {
			stepDef.SetSudo()
		}

		result = append(result, stepDef)
	}

	return result, nil
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

// injectTriggeredRunID returns a copy of steps where any actions/download-artifact
// step that doesn't already have run-id set gets it injected with the workflow_run
// event's run ID. This avoids mutating the shared jobDef.Steps slice.
func injectTriggeredRunID(steps []*ghworkflow.JobStep) []*ghworkflow.JobStep {
	downloadRef := "actions/download-artifact@" + config.DownloadArtifactActionRef
	result := make([]*ghworkflow.JobStep, len(steps))

	for i, step := range steps {
		if step.Uses.Image == downloadRef {
			if _, ok := step.With["run-id"]; !ok {
				clone := *step
				clone.With = make(map[string]string, len(step.With)+1)

				maps.Copy(clone.With, step.With)

				clone.With["run-id"] = "${{ github.event.workflow_run.id }}"
				clone.With["github-token"] = "${{ secrets.GITHUB_TOKEN }}"
				result[i] = &clone

				continue
			}
		}

		result[i] = step
	}

	return result
}

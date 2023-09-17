// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package ghworkflow implements output to .github/workflows/ci.yaml.
package ghworkflow

import (
	_ "embed"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/output"
)

const (
	hostedRunner  = "self-hosted"
	ciWorkflow    = ".github/workflows/ci.yaml"
	slackWorkflow = ".github/workflows/slack-notify.yaml"
)

//go:embed files/buildkitd.toml
var buildkitdConfig string

//go:embed files/slack-notify-payload.json
var slackNotifyPayload string

// Output implements GitHub Actions project config generation.
type Output struct {
	output.FileAdapter

	defaultWorkflow     *Workflow
	slackNotifyWorkflow *Workflow
}

// NewOutput creates new .github/workflows/ci.yaml output.
func NewOutput() *Output {
	output := &Output{
		defaultWorkflow: &Workflow{
			Name: "default",
			On: On{
				Push: Push{
					Branches: []string{
						"main",
						"release-*",
					},
					Tags: []string{"v*"},
				},
				PullRequest: PullRequest{
					Branches: []string{
						"main",
						"release-*",
					},
				},
			},
			Jobs: map[string]*Job{
				"default": {
					If:     "${{ !startsWith(github.head_ref, 'renovate/') || !startsWith(github.head_ref, 'renovate/') }}",
					RunsOn: []string{hostedRunner, "X64"},
					Permissions: map[string]string{
						"packages": "write",
						"contents": "write",
					},
					Steps: DefaultSteps(),
				},
			},
		},
		slackNotifyWorkflow: &Workflow{
			Name: "slack-notify",
			On: On{
				WorkFlowRun: WorkFlowRun{
					Workflows: []string{"default"},
					Types:     []string{"completed"},
				},
			},
			Jobs: map[string]*Job{
				"slack-notify": {
					RunsOn: []string{hostedRunner},
					Steps: []*Step{
						{
							Name: "Retrieve Workflow Run Info",
							ID:   "retrieve-workflow-run-info",
							Uses: fmt.Sprintf("potiuk/get-workflow-origin@%s", config.GetWorkflowOriginActionVersion),
							With: map[string]string{
								"token":       "${{ secrets.GITHUB_TOKEN }}",
								"sourceRunId": "${{ github.event.workflow_run.id }}",
							},
						},
						{
							Name: "Slack Notify",
							Uses: fmt.Sprintf("slackapi/slack-github-action@%s", config.SlackNotifyActionVersion),
							With: map[string]string{
								"channel-id": "proj-talos-maintainers",
								"payload":    slackNotifyPayload,
							},
							Env: map[string]string{
								"SLACK_BOT_TOKEN": "${{ secrets.SLACK_BOT_TOKEN }}",
							},
						},
					},
				},
			},
		},
	}

	output.FileWriter = output

	return output
}

// AddJob adds job to the workflow.
func (o *Output) AddJob(name string, job *Job) {
	o.defaultWorkflow.Jobs[name] = job
}

// AddStep adds step to the job.
func (o *Output) AddStep(jobName string, steps ...*Step) {
	o.defaultWorkflow.Jobs[jobName].Steps = append(o.defaultWorkflow.Jobs[jobName].Steps, steps...)
}

// DefaultSteps returns default steps for the workflow.
func DefaultSteps() []*Step {
	return []*Step{
		{
			Name: "checkout",
			Uses: fmt.Sprintf("actions/checkout@%s", config.CheckOutActionVersion),
		},
		{
			Name: "Unshallow",
			Run:  "git fetch --prune --unshallow\n",
		},
		{
			Name: "Set up Docker Buildx",
			Uses: fmt.Sprintf("docker/setup-buildx-action@%s", config.SetupBuildxActionVersion),
			With: map[string]string{
				"config-inline": buildkitdConfig,
			},
		},
	}
}

// MakeStep creates a step with make command.
func MakeStep(name string, args ...string) *Step {
	command := fmt.Sprintf("make %s\n", name)

	if len(args) > 0 {
		command = fmt.Sprintf("make %s %s\n", name, strings.Join(args, " "))
	}

	return &Step{
		Name: name,
		Run:  command,
	}
}

// SetName sets step name.
func (step *Step) SetName(name string) *Step {
	step.Name = name

	return step
}

// SetEnv sets step environment variables.
func (step *Step) SetEnv(name, value string) *Step {
	if step.Env == nil {
		step.Env = map[string]string{}
	}

	step.Env[name] = value

	return step
}

// ExceptPullRequest adds condition to skip step on PRs.
func (step *Step) ExceptPullRequest() *Step {
	step.If = "github.event_name != 'pull_request'"

	return step
}

// OnlyOnTag adds condition to run step only on tags.
func (step *Step) OnlyOnTag() *Step {
	step.If = "startsWith(github.ref, 'refs/tags/')"

	return step
}

// Compile implements [output.TypedWriter] interface.
func (o *Output) Compile(compiler Compiler) error {
	return compiler.CompileGitHubWorkflow(o)
}

// Filenames implements output.FileWriter interface.
func (o *Output) Filenames() []string {
	return []string{ciWorkflow, slackWorkflow}
}

// GenerateFile implements output.FileWriter interface.
func (o *Output) GenerateFile(filename string, w io.Writer) error {
	switch filename {
	case ciWorkflow:
		return o.ghWorkflow(w)
	case slackWorkflow:
		return o.slackWorkflow(w)
	default:
		panic("unexpected filename: " + filename)
	}
}

func (o *Output) ghWorkflow(w io.Writer) error {
	preamble := output.Preamble("# ")

	if _, err := w.Write([]byte(preamble)); err != nil {
		return fmt.Errorf("failed to write preamble: %w", err)
	}

	encoder := yaml.NewEncoder(w)

	defer encoder.Close() //nolint:errcheck

	encoder.SetIndent(2)

	if err := encoder.Encode(o.defaultWorkflow); err != nil {
		return fmt.Errorf("failed to encode workflow: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return fmt.Errorf("failed to close encoder: %w", err)
	}

	return nil
}

func (o *Output) slackWorkflow(w io.Writer) error {
	preamble := output.Preamble("# ")

	if _, err := w.Write([]byte(preamble)); err != nil {
		return fmt.Errorf("failed to write preamble: %w", err)
	}

	encoder := yaml.NewEncoder(w)

	defer encoder.Close() //nolint:errcheck

	encoder.SetIndent(2)

	if err := encoder.Encode(o.slackNotifyWorkflow); err != nil {
		return fmt.Errorf("failed to encode workflow: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return fmt.Errorf("failed to close encoder: %w", err)
	}

	return nil
}

// Compiler is implemented by project blocks which support GitHub Actions config generation.
type Compiler interface {
	CompileGitHubWorkflow(*Output) error
}

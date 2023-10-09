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

	"github.com/siderolabs/gen/maps"
	"gopkg.in/yaml.v3"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/output"
)

const (
	// HostedRunner is the name of the hosted runner.
	HostedRunner = "self-hosted"
	// GenericRunner is the name of the generic runner.
	GenericRunner = "generic"
	// DefaultSkipCondition is the default condition to skip the workflow.
	DefaultSkipCondition = "(!startsWith(github.head_ref, 'renovate/') && !startsWith(github.head_ref, 'dependabot/')) && contains(github.event.pull_request.labels.*.name, 'status/ok-to-test')"

	workflowDir   = ".github/workflows"
	ciWorkflow    = workflowDir + "/" + "ci.yaml"
	slackWorkflow = workflowDir + "/" + "slack-notify.yaml"
)

//go:embed files/slack-notify-payload.json
var slackNotifyPayload string

// Output implements GitHub Actions project config generation.
type Output struct {
	output.FileAdapter

	workflows map[string]*Workflow
}

// NewOutput creates new .github/workflows/ci.yaml output.
func NewOutput() *Output {
	output := &Output{
		workflows: map[string]*Workflow{
			ciWorkflow: {
				Name: "default",
				// https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#example-using-a-fallback-value
				Concurrency: Concurrency{
					Group:            "${{ github.event.label == null && github.head_ref || github.run_id }}",
					CancelInProgress: true,
				},
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
						If: DefaultSkipCondition,
						RunsOn: []string{
							HostedRunner,
							GenericRunner,
						},
						Permissions: map[string]string{
							"packages": "write",
							"contents": "write",
						},
						Services: DefaultServices(),
						Steps:    DefaultSteps(),
					},
				},
			},
			slackWorkflow: {
				Name: "slack-notify",
				On: On{
					WorkFlowRun: WorkFlowRun{
						Workflows: []string{"default"},
						Types:     []string{"completed"},
					},
				},
				Jobs: map[string]*Job{
					"slack-notify": {
						RunsOn: []string{
							HostedRunner,
							GenericRunner,
						},
						If: "${{ github.event.workflow_run.conclusion != 'skipped' }}",
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
		},
	}

	output.FileWriter = output

	return output
}

// AddWorkflow adds workflow to the output.
func (o *Output) AddWorkflow(name string, workflow *Workflow) {
	file := workflowDir + "/" + name + ".yaml"

	switch file {
	case ciWorkflow, slackWorkflow:
		panic(fmt.Sprintf("workflow %s is reserved", file))
	}

	o.workflows[file] = workflow
}

// AddJob adds job to the default workflow.
func (o *Output) AddJob(name string, job *Job) {
	o.workflows[ciWorkflow].Jobs[name] = job
}

// AddStep adds step to the job.
func (o *Output) AddStep(jobName string, steps ...*Step) {
	o.workflows[ciWorkflow].Jobs[jobName].Steps = append(o.workflows[ciWorkflow].Jobs[jobName].Steps, steps...)
}

// AddSlackNotify adds the workflow to notify slack dependencies.
func (o *Output) AddSlackNotify(workflow string) {
	o.workflows[slackWorkflow].Workflows = append(o.workflows[slackWorkflow].Workflows, workflow)
}

// OverrideDefaultJobCondition overrides default job condition.
func (o *Output) OverrideDefaultJobCondition(condition string) {
	o.workflows[ciWorkflow].Jobs["default"].If = condition
}

// AddPullRequestLabelCondition adds condition to default job to also run on PRs with labels.
func (o *Output) AddPullRequestLabelCondition() {
	o.workflows[ciWorkflow].On.PullRequest.Types = []string{
		"opened",
		"synchronize",
		"reopened",
		"labeled",
	}
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
				"driver":   "remote",
				"endpoint": "tcp://localhost:1234",
			},
		},
	}
}

// DefaultServices returns default services for the workflow.
func DefaultServices() map[string]Service {
	return map[string]Service{
		"buildkitd": {
			Image:   fmt.Sprintf("moby/buildkit:%s", config.BuildKitContainerVersion),
			Options: "--privileged",
			Ports:   []string{"1234:1234"},
			Volumes: []string{
				"/var/lib/buildkit/${{ github.repository }}:/var/lib/buildkit",
				"/usr/etc/buildkit/buildkitd.toml:/etc/buildkit/buildkitd.toml",
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
	return maps.Keys(o.workflows)
}

// GenerateFile implements output.FileWriter interface.
func (o *Output) GenerateFile(filename string, w io.Writer) error {
	return o.ghWorkflow(w, filename)
}

func (o *Output) ghWorkflow(w io.Writer, name string) error {
	preamble := output.Preamble("# ")

	if _, err := w.Write([]byte(preamble)); err != nil {
		return fmt.Errorf("failed to write preamble: %w", err)
	}

	encoder := yaml.NewEncoder(w)

	defer encoder.Close() //nolint:errcheck

	encoder.SetIndent(2)

	if err := encoder.Encode(o.workflows[name]); err != nil {
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

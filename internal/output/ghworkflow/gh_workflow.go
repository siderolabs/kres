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
	// PkgsRunner is the name of the default runner for packages.
	PkgsRunner = "pkgs"
	// DefaultSkipCondition is the default condition to skip the workflow.
	DefaultSkipCondition = "(!startsWith(github.head_ref, 'renovate/') && !startsWith(github.head_ref, 'dependabot/'))"

	// IssueLabelRetrieveScript is the default script to retrieve issue labels.
	IssueLabelRetrieveScript = `
if (context.eventName != "pull_request") { return "[]" }

const resp = await github.rest.issues.get({
    issue_number: context.issue.number,
    owner: context.repo.owner,
    repo: context.repo.repo,
})

return resp.data.labels.map(label => label.name)
`

	workflowDir   = ".github/workflows"
	ciWorkflow    = workflowDir + "/" + "ci.yaml"
	slackWorkflow = workflowDir + "/" + "slack-notify.yaml"
)

var (
	//go:embed files/slack-notify-payload.json
	slackNotifyPayload string

	armbuildkitdEnpointConfig = `
- endpoint: tcp://buildkit-arm64.ci.svc.cluster.local:1234
  platforms: linux/arm64
`
)

// Output implements GitHub Actions project config generation.
type Output struct {
	output.FileAdapter

	workflows map[string]*Workflow
}

// NewOutput creates new .github/workflows/ci.yaml output.
func NewOutput(mainBranch string) *Output {
	output := &Output{
		workflows: map[string]*Workflow{
			ciWorkflow: {
				Name: "default",
				// https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#example-using-a-fallback-value
				Concurrency: Concurrency{
					Group:            "${{ github.head_ref || github.run_id }}",
					CancelInProgress: true,
				},
				On: On{
					Push: Push{
						Branches: []string{
							mainBranch,
							"release-*",
						},
						Tags: []string{"v*"},
					},
					PullRequest: PullRequest{
						Branches: []string{
							mainBranch,
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
							"packages":      "write",
							"contents":      "write",
							"actions":       "read",
							"pull-requests": "read",
							"issues":        "read",
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
						If: "github.event.workflow_run.conclusion != 'skipped'",
						Steps: []*Step{
							{
								Name: "Get PR number",
								ID:   "get-pr-number",
								If:   "github.event.workflow_run.event == 'pull_request'",
								Env: map[string]string{
									"GH_TOKEN": "${{ github.token }}",
								},
								Run: "echo pull_request_number=$(gh pr view -R ${{ github.repository }} ${{ github.event.workflow_run.head_repository.owner.login }}:${{ github.event.workflow_run.head_branch }} --json number --jq .number) >> $GITHUB_OUTPUT\n", //nolint:lll
							},
							{
								Name: "Slack Notify",
								Uses: "slackapi/slack-github-action@" + config.SlackNotifyActionVersion,
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

// AddOutputs adds outputs to the job.
func (o *Output) AddOutputs(jobName string, outputs map[string]string) {
	o.workflows[ciWorkflow].Jobs[jobName].Outputs = outputs
}

// AddSlackNotify adds the workflow to notify slack dependencies.
func (o *Output) AddSlackNotify(workflow string) {
	o.workflows[slackWorkflow].Workflows = append(o.workflows[slackWorkflow].Workflows, workflow)
}

// SetDefaultJobRunnerAsPkgs sets default job runner as pkgs.
func (o *Output) SetDefaultJobRunnerAsPkgs() {
	o.workflows[ciWorkflow].Jobs["default"].RunsOn = []string{
		HostedRunner,
		PkgsRunner,
	}
}

// OverwriteDefaultJobStepsAsPkgs overwrites default job steps as pkgs.
// Note that calling this method will overwrite any existing steps.
func (o *Output) OverwriteDefaultJobStepsAsPkgs() {
	o.workflows[ciWorkflow].Jobs["default"].Steps = DefaultPkgsSteps()
}

// CommonSteps returns common steps for the workflow.
func CommonSteps() []*Step {
	return []*Step{
		{
			Name: "checkout",
			Uses: "actions/checkout@" + config.CheckOutActionVersion,
		},
		{
			Name: "Unshallow",
			Run:  "git fetch --prune --unshallow\n",
		},
	}
}

// DefaultSteps returns default steps for the workflow.
func DefaultSteps() []*Step {
	return append(
		CommonSteps(),
		&Step{
			Name: "Set up Docker Buildx",
			Uses: "docker/setup-buildx-action@" + config.SetupBuildxActionVersion,
			With: map[string]string{
				"driver":   "remote",
				"endpoint": "tcp://127.0.0.1:1234",
			},
			TimeoutMinutes: 10,
		},
	)
}

// DefaultPkgsSteps returns default pkgs steps for the workflow.
func DefaultPkgsSteps() []*Step {
	return append(
		CommonSteps(),
		&Step{
			Name: "Set up Docker Buildx",
			Uses: "docker/setup-buildx-action@" + config.SetupBuildxActionVersion,
			With: map[string]string{
				"driver":   "remote",
				"endpoint": "tcp://127.0.0.1:1234",
				"append":   strings.TrimPrefix(armbuildkitdEnpointConfig, "\n"),
			},
		},
	)
}

// DefaultServices returns default services for the workflow.
func DefaultServices() map[string]Service {
	return map[string]Service{
		"buildkitd": {
			Image:   "moby/buildkit:" + config.BuildKitContainerVersion,
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

	if name == "" {
		command = "make\n"
	}

	if len(args) > 0 {
		command = fmt.Sprintf("make %s %s\n", name, strings.Join(args, " "))
	}

	return &Step{
		Name: name,
		Run:  command,
	}
}

// SetSudo sets step to run with sudo.
func (step *Step) SetSudo() *Step {
	step.Run = "sudo -E " + step.Run

	return step
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

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package ghworkflow implements output to .github/workflows/ci.yaml.
package ghworkflow

import (
	_ "embed"
	"fmt"
	"io"
	"slices"
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
	// SystemInfoPrintScript is the script to print system info.
	SystemInfoPrintScript = `
MEMORY_GB=$((${{ steps.system-info.outputs.totalmem }}/1024/1024/1024))

OUTPUTS=(
  "CPU Core: ${{ steps.system-info.outputs.cpu-core }}"
  "CPU Model: ${{ steps.system-info.outputs.cpu-model }}"
  "Hostname: ${{ steps.system-info.outputs.hostname }}"
  "NodeName: ${NODE_NAME}"
  "Kernel release: ${{ steps.system-info.outputs.kernel-release }}"
  "Kernel version: ${{ steps.system-info.outputs.kernel-version }}"
  "Name: ${{ steps.system-info.outputs.name }}"
  "Platform: ${{ steps.system-info.outputs.platform }}"
  "Release: ${{ steps.system-info.outputs.release }}"
  "Total memory: ${MEMORY_GB} GB"
)

for OUTPUT in "${OUTPUTS[@]}";do
  echo "${OUTPUT}"
done
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
func NewOutput(mainBranch string, withDefaultJob bool, withStaleJob bool) *Output {
	workflows := map[string]*Workflow{
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
					Steps: []*JobStep{
						Step("Get PR number").
							SetID("get-pr-number").
							SetEnv("GH_TOKEN", "${{ github.token }}").
							SetCommand("echo pull_request_number=$(gh pr view -R ${{ github.repository }} ${{ github.event.workflow_run.head_repository.owner.login }}:${{ github.event.workflow_run.head_branch }} --json number --jq .number) >> $GITHUB_OUTPUT"). //nolint:lll
							SetCustomCondition("github.event.workflow_run.event == 'pull_request'"),
						Step("Slack Notify").
							SetUses("slackapi/slack-github-action@"+config.SlackNotifyActionVersion).
							SetWith("token", "${{ secrets.SLACK_BOT_TOKEN_V2 }}").
							SetWith("method", "chat.postMessage").
							SetWith("payload", slackNotifyPayload),
					},
				},
			},
		},
	}

	if withStaleJob {
		workflows[".github/workflows/lock.yml"] = &Workflow{
			Name: "Lock old issues",
			On: On{
				Schedule: []Schedule{
					{
						Cron: "0 2 * * *", // Every day at 2 AM
					},
				},
			},
			Permissions: map[string]string{
				"issues": "write",
			},
			Jobs: map[string]*Job{
				"action": {
					RunsOn: []string{"ubuntu-latest"},
					Steps: []*JobStep{
						{
							Name: "Lock old issues",
							Uses: "dessant/lock-threads@" + config.LockThreadsActionVersion,
							With: map[string]string{
								"issue-inactive-days": "60",
								"process-only":        "issues",
								"log-output":          "true",
							},
						},
					},
				},
			},
		}

		workflows[".github/workflows/stale.yml"] = &Workflow{
			Name: "Close stale issues and PRs",
			On: On{
				Schedule: []Schedule{
					{
						Cron: "30 1 * * *", // Every day at 1:30 AM
					},
				},
			},
			Permissions: map[string]string{
				"issues":        "write",
				"pull-requests": "write",
			},
			Jobs: map[string]*Job{
				"stale": {
					RunsOn: []string{"ubuntu-latest"},
					Steps: []*JobStep{
						{
							Name: "Close stale issues and PRs",
							Uses: "actions/stale@" + config.StaleActionVersion,
							With: map[string]string{
								"stale-issue-message":     "This issue is stale because it has been open 180 days with no activity. Remove stale label or comment or this will be closed in 7 days.",
								"stale-pr-message":        "This PR is stale because it has been open 45 days with no activity.",
								"close-issue-message":     "This issue was closed because it has been stalled for 7 days with no activity.",
								"days-before-issue-stale": "180",
								"days-before-pr-stale":    "45",
								"days-before-issue-close": "5",
								"days-before-pr-close":    "-1",   // never close PRs
								"operations-per-run":      "2000", // the maximum number of operations to perform per run
							},
						},
					},
				},
			},
		}
	}

	if withDefaultJob {
		workflows[ciWorkflow].Jobs = map[string]*Job{
			"default": {
				If:          DefaultSkipCondition,
				Permissions: DefaultJobPermissions(),
				Steps:       DefaultSteps(),
			},
		}
	}

	output := &Output{
		workflows: workflows,
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
	if o.workflows[ciWorkflow].Jobs == nil {
		o.workflows[ciWorkflow].Jobs = map[string]*Job{}
	}

	o.workflows[ciWorkflow].Jobs[name] = job
}

// AddStep adds step to the job.
func (o *Output) AddStep(jobName string, steps ...*JobStep) {
	o.workflows[ciWorkflow].Jobs[jobName].Steps = append(o.workflows[ciWorkflow].Jobs[jobName].Steps, steps...)
}

// CheckIfStepExists checks if step with given ID exists in the job.
func (o *Output) CheckIfStepExists(jobName, stepID string) bool {
	job := o.workflows[ciWorkflow].Jobs[jobName]

	if job == nil {
		return false
	}

	return slices.ContainsFunc(job.Steps, func(s *JobStep) bool { return s.ID == stepID })
}

// AddJobPermissions adds permissions to the job.
func (o *Output) AddJobPermissions(jobName, permission, value string) {
	o.workflows[ciWorkflow].Jobs[jobName].Permissions[permission] = value
}

// AddStepBefore adds step before another step in the job.
func (o *Output) AddStepBefore(jobName, beforeStepID string, steps ...*JobStep) {
	job := o.workflows[ciWorkflow].Jobs[jobName]

	idx := slices.IndexFunc(job.Steps, func(s *JobStep) bool { return s.ID == beforeStepID })
	if idx != -1 {
		job.Steps = slices.Insert(job.Steps, idx, steps...)
	}
}

// AddStepAfter adds step after another step in the job.
func (o *Output) AddStepAfter(jobName, afterStepID string, steps ...*JobStep) {
	job := o.workflows[ciWorkflow].Jobs[jobName]

	if job == nil {
		return
	}

	idx := slices.IndexFunc(job.Steps, func(s *JobStep) bool { return s.ID == afterStepID })

	if idx != -1 {
		job.Steps = slices.Insert(job.Steps, idx+1, steps...)
	}
}

// AddOutputs adds outputs to the job.
func (o *Output) AddOutputs(jobName string, outputs map[string]string) {
	o.workflows[ciWorkflow].Jobs[jobName].Outputs = outputs
}

// AddSlackNotify adds the workflow to notify slack dependencies.
func (o *Output) AddSlackNotify(workflow string) {
	o.workflows[slackWorkflow].Workflows = append(o.workflows[slackWorkflow].Workflows, workflow)
}

// SetRunners allows to set custom runners for the default job.
// If runners are not provided, the default runners will be used.
func (o *Output) SetRunners(runners ...string) {
	if len(runners) == 0 {
		o.workflows[ciWorkflow].Jobs["default"].RunsOn = []string{
			HostedRunner,
			GenericRunner,
		}

		return
	}

	o.workflows[ciWorkflow].Jobs["default"].RunsOn = runners
}

// SetOptionsForPkgs overwrites default job steps and services for pkgs.
// Note that calling this method will overwrite any existing steps.
func (o *Output) SetOptionsForPkgs() {
	o.SetRunners(HostedRunner, PkgsRunner)

	o.workflows[ciWorkflow].Jobs["default"].Steps = DefaultPkgsSteps()
}

// CommonSteps returns common steps for the workflow.
func CommonSteps() []*JobStep {
	return []*JobStep{
		Step("gather-system-info").
			SetUses("kenchan0130/actions-system-info@" + config.SystemInfoActionVersion).
			SetID("system-info").
			SetContinueOnError(),
		Step("print-system-info").
			SetCommand(strings.Trim(SystemInfoPrintScript, "\n")).
			SetContinueOnError(),
		Step("checkout").
			SetUses("actions/checkout@" + config.CheckOutActionVersion),
		Step("Unshallow").
			SetCommand("git fetch --prune --unshallow"),
	}
}

// DefaultJobPermissions returns default job permissions.
func DefaultJobPermissions() map[string]string {
	return map[string]string{
		"packages":      "write",
		"contents":      "write",
		"actions":       "read",
		"pull-requests": "read",
		"issues":        "read",
	}
}

// DefaultSteps returns default steps for the workflow.
func DefaultSteps() []*JobStep {
	return append(
		CommonSteps(),
		&JobStep{
			Name: "Set up Docker Buildx",
			ID:   "setup-buildx",
			Uses: "docker/setup-buildx-action@" + config.SetupBuildxActionVersion,
			With: map[string]string{
				"driver":   "remote",
				"endpoint": "tcp://buildkit-amd64.ci.svc.cluster.local:1234",
			},
			TimeoutMinutes: 10,
		},
	)
}

// DefaultPkgsSteps returns default pkgs steps for the workflow.
func DefaultPkgsSteps() []*JobStep {
	return append(
		CommonSteps(),
		&JobStep{
			Name: "Set up Docker Buildx",
			ID:   "setup-buildx",
			Uses: "docker/setup-buildx-action@" + config.SetupBuildxActionVersion,
			With: map[string]string{
				"driver":   "remote",
				"endpoint": "tcp://buildkit-amd64.ci.svc.cluster.local:1234",
				"append":   strings.TrimPrefix(armbuildkitdEnpointConfig, "\n"),
			},
		},
	)
}

// SOPSSteps returns SOPS steps for the workflow.
func SOPSSteps() []*JobStep {
	return []*JobStep{
		{
			Name: "Mask secrets",
			Run:  "echo \"$(sops -d .secrets.yaml | yq -e '.secrets | to_entries[] | \"::add-mask::\" + .value')\"\n",
		},
		{
			Name: "Set secrets for job",
			Run:  "sops -d .secrets.yaml | yq -e '.secrets | to_entries[] | .key + \"=\" + .value' >> \"$GITHUB_ENV\"\n",
		},
	}
}

// Step creates a step with name.
func Step(name string) *JobStep {
	return &JobStep{
		Name: name,
	}
}

// SetUses sets step to use action.
func (step *JobStep) SetUses(uses string) *JobStep {
	step.Uses = uses

	return step
}

// SetMakeStep sets step to run make command.
func (step *JobStep) SetMakeStep(target string, args ...string) *JobStep {
	command := fmt.Sprintf("make %s", target)

	if target == "" {
		command = "make"
	}

	if len(args) > 0 {
		command = fmt.Sprintf("make %s %s", target, strings.Join(args, " "))
	}

	return step.SetCommand(command)
}

// SetSudo sets step to run with sudo.
func (step *JobStep) SetSudo() *JobStep {
	step.Run = "sudo -E " + step.Run

	return step
}

// SetCommand sets step command.
func (step *JobStep) SetCommand(command string) *JobStep {
	step.Run = command + "\n"

	return step
}

// SetEnv sets step environment variables.
func (step *JobStep) SetEnv(name, value string) *JobStep {
	if step.Env == nil {
		step.Env = map[string]string{}
	}

	step.Env[name] = value

	return step
}

// SetTimeoutMinutes sets step timeout in minutes.
func (step *JobStep) SetTimeoutMinutes(minutes int) *JobStep {
	step.TimeoutMinutes = minutes

	return step
}

// SetContinueOnError sets step to continue on error.
func (step *JobStep) SetContinueOnError() *JobStep {
	step.ContinueOnError = true

	return step
}

// SetWith sets step with key and value.
func (step *JobStep) SetWith(key, value string) *JobStep {
	if step.With == nil {
		step.With = map[string]string{}
	}

	step.With[key] = value

	return step
}

// SetID sets step ID.
func (step *JobStep) SetID(id string) *JobStep {
	step.ID = id

	return step
}

func (step *JobStep) appendIf(condition string) {
	if step.If == "" {
		step.If = condition
	} else {
		step.If += " && " + condition
	}
}

// SetCustomCondition sets a custom condition clearing out any previously set conditions.
func (step *JobStep) SetCustomCondition(condition string) *JobStep {
	step.If = condition

	return step
}

// SetConditionOnlyOnBranch adds condition to run step only on a specific branch name.
func (step *JobStep) SetConditionOnlyOnBranch(name string) *JobStep {
	step.appendIf(fmt.Sprintf("github.ref == 'refs/heads/%s'", name))

	return step
}

// SetConditions sets step conditions.
func (step *JobStep) SetConditions(conditions ...string) error {
	for _, condition := range conditions {
		switch condition {
		case "except-pull-request":
			step.appendIf("github.event_name != 'pull_request'")
		case "on-pull-request":
			step.appendIf("github.event_name == 'pull_request'")
		case "only-on-tag":
			step.appendIf("startsWith(github.ref, 'refs/tags/')")
		case "not-on-tag":
			step.appendIf("!startsWith(github.ref, 'refs/tags/')")
		case "only-on-schedule":
			step.appendIf("github.event_name == 'schedule'")
		case "not-on-schedule":
			step.appendIf("github.event_name != 'schedule'")
		case "always":
			step.appendIf("always()")
		case "":
			return nil
		default:
			return fmt.Errorf("unknown condition: %s", condition)
		}
	}

	return nil
}

func (job *Job) appendIf(condition string) {
	if job.If == "" {
		job.If = condition
	} else {
		job.If += " && " + condition
	}
}

// SetConditions sets job conditions.
func (job *Job) SetConditions(conditions ...string) error {
	for _, condition := range conditions {
		switch condition {
		case "except-pull-request":
			job.appendIf("github.event_name != 'pull_request'")
		case "on-pull-request":
			job.appendIf("github.event_name == 'pull_request'")
		case "only-on-tag":
			job.appendIf("startsWith(github.ref, 'refs/tags/')")
		case "not-on-tag":
			job.appendIf("!startsWith(github.ref, 'refs/tags/')")
		case "only-on-schedule":
			job.appendIf("github.event_name == 'schedule'")
		case "not-on-schedule":
			job.appendIf("github.event_name != 'schedule'")
		case "always":
			job.appendIf("always()")
		case "":
			return nil
		default:
			return fmt.Errorf("unknown condition: %s", condition)
		}
	}

	return nil
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

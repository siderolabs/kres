// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package ghworkflow

// Workflow represents Github Actions workflow.
//
//nolint:govet
type Workflow struct {
	Name        string `yaml:"name"`
	Concurrency `yaml:"concurrency,omitempty"`
	On          `yaml:"on"`
	Env         map[string]string `yaml:"env,omitempty"`
	Jobs        map[string]*Job   `yaml:"jobs"`
}

// Concurrency represents GitHub Actions concurrency.
type Concurrency struct {
	Group            string `yaml:"group"`
	CancelInProgress bool   `yaml:"cancel-in-progress"`
}

// On represents GitHub Actions event triggers.
type On struct {
	Push        `yaml:"push,omitempty"`
	PullRequest `yaml:"pull_request,omitempty"`
	Schedule    []Schedule `yaml:"schedule,omitempty"`
	WorkFlowRun `yaml:"workflow_run,omitempty"`
}

// Branches represents GitHub Actions branch filters.
type Branches []string

// PullRequest represents GitHub Actions pull request filters.
type PullRequest struct {
	Branches `yaml:"branches,omitempty"`
	Types    []string `yaml:"types,omitempty"`
}

// Schedule represents GitHub Actions schedule filters.
type Schedule struct {
	Cron string `yaml:"cron"`
}

// WorkFlowRun represents GitHub Actions workflow_run filters.
type WorkFlowRun struct {
	Workflows []string `yaml:"workflows"`
	Types     []string `yaml:"types"`
}

// PullRequestTarget represents GitHub Actions pull request target filters.
type PullRequestTarget struct{}

// Push represents GitHub Actions push filters.
type Push struct {
	Branches `yaml:"branches"`
	Tags     []string `yaml:"tags,omitempty"`
}

// Job represents GitHub Actions job.
type Job struct {
	Permissions map[string]string  `yaml:"permissions,omitempty"`
	RunsOn      []string           `yaml:"runs-on"`
	If          string             `yaml:"if,omitempty"`
	Needs       []string           `yaml:"needs,omitempty"`
	Outputs     map[string]string  `yaml:"outputs,omitempty"`
	Services    map[string]Service `yaml:"services,omitempty"`
	Steps       []*JobStep         `yaml:"steps"`
}

// Service represents GitHub Actions service.
type Service struct {
	Image   string   `yaml:"image"`
	Options string   `yaml:"options,omitempty"`
	Ports   []string `yaml:"ports,omitempty"`
	Volumes []string `yaml:"volumes,omitempty"`
}

// JobStep represents GitHub Actions job step.
type JobStep struct {
	Name            string            `yaml:"name"`
	ID              string            `yaml:"id,omitempty"`
	If              string            `yaml:"if,omitempty"`
	Uses            string            `yaml:"uses,omitempty"`
	With            map[string]string `yaml:"with,omitempty"`
	Env             map[string]string `yaml:"env,omitempty"`
	Run             string            `yaml:"run,omitempty"`
	ContinueOnError bool              `yaml:"continue-on-error,omitempty"`
	TimeoutMinutes  int               `yaml:"timeout-minutes,omitempty"`
}

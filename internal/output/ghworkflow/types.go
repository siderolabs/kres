// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package ghworkflow

// Workflow represents Github Actions workflow.
//
//nolint:govet
type Workflow struct {
	Name string            `yaml:"name"`
	On   On                `yaml:"on"`
	Env  map[string]string `yaml:"env,omitempty"`
	Jobs map[string]*Job   `yaml:"jobs"`
}

// On represents GitHub Actions event triggers.
type On struct {
	Push        `yaml:"push"`
	PullRequest `yaml:"pull_request"`
}

// Branches represents GitHub Actions branch filters.
type Branches []string

// PullRequest represents GitHub Actions pull request filters.
type PullRequest struct {
	Branches `yaml:"branches,omitempty"`
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
	Permissions map[string]string `yaml:"permissions,omitempty"`
	RunsOn      []string          `yaml:"runs-on"`
	If          string            `yaml:"if,omitempty"`
	Needs       []string          `yaml:"needs,omitempty"`
	Steps       []*Step           `yaml:"steps"`
}

// Step represents GitHub Actions step.
type Step struct {
	Name string            `yaml:"name"`
	If   string            `yaml:"if,omitempty"`
	Uses string            `yaml:"uses,omitempty"`
	With map[string]string `yaml:"with,omitempty"`
	Env  map[string]string `yaml:"env,omitempty"`
	Run  string            `yaml:"run,omitempty"`
}

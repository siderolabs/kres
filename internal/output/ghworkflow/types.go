// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package ghworkflow

import (
	"fmt"

	"go.yaml.in/yaml/v4"
)

// Workflow represents Github Actions workflow.
//
//nolint:govet
type Workflow struct {
	Concurrency `yaml:"concurrency,omitempty"`
	On          `yaml:"on"`

	Name        string            `yaml:"name"`
	Permissions map[string]string `yaml:"permissions,omitempty"`
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
	Push              `yaml:"push,omitempty"`
	PullRequest       `yaml:"pull_request,omitempty"`
	WorkFlowRun       `yaml:"workflow_run,omitempty"`
	*WorkFlowDispatch `yaml:"workflow_dispatch,omitempty"`

	Schedule []Schedule `yaml:"schedule,omitempty"`
}

// Branches represents GitHub Actions branch filters.
type Branches []string

// PullRequest represents GitHub Actions pull request filters.
type PullRequest struct {
	Branches `yaml:"branches,omitempty"`

	Types []string `yaml:"types,omitempty"`
	Paths []string `yaml:"paths,omitempty"`
}

// Schedule represents GitHub Actions schedule filters.
type Schedule struct {
	Cron string `yaml:"cron"`
}

// WorkFlowRun represents GitHub Actions workflow_run filters.
type WorkFlowRun struct {
	Workflows []string `yaml:"workflows"`
	Types     []string `yaml:"types"`
	Branches  []string `yaml:"branches,omitempty"`
}

// WorkFlowDispatch represents GitHub Actions workflow_dispatch filters.
type WorkFlowDispatch struct {
	Inputs map[string]WorkFlowDispatchInput `yaml:"inputs,omitempty"`
}

// WorkFlowDispatchInput represents a single input for workflow_dispatch.
type WorkFlowDispatchInput struct {
	Description string   `yaml:"description"`
	Default     string   `yaml:"default,omitempty"`
	Type        string   `yaml:"type"`
	Options     []string `yaml:"options,omitempty"`
	Required    bool     `yaml:"required"`
}

// PullRequestTarget represents GitHub Actions pull request target filters.
type PullRequestTarget struct{}

// Push represents GitHub Actions push filters.
type Push struct {
	Branches `yaml:"branches,omitempty"`

	Tags []string `yaml:"tags,omitempty"`
}

// Job represents GitHub Actions job.
type Job struct {
	Permissions map[string]string  `yaml:"permissions,omitempty"`
	RunsOn      RunsOn             `yaml:"runs-on"`
	If          string             `yaml:"if,omitempty"`
	Needs       []string           `yaml:"needs,omitempty"`
	Outputs     map[string]string  `yaml:"outputs,omitempty"`
	Services    map[string]Service `yaml:"services,omitempty"`
	Steps       []*JobStep         `yaml:"steps"`
}

// RunsOn represents GitHub Actions runs-on field which can be a string, slice, or type with Group/Label structure.
type RunsOn struct {
	value any
}

type RunsOnGroupLabel struct {
	Group string `yaml:"group,omitempty"`
	Label string `yaml:"label,omitempty"`
}

// MarshalYAML implements yaml.Marshaler.
func (r RunsOn) MarshalYAML() (any, error) {
	// check if r.value is nil or empty
	if r.value == nil {
		return nil, fmt.Errorf("runs-on needs to be set")
	}

	// check for empty values based on type
	switch v := r.value.(type) {
	case string:
		if v == "" {
			return nil, fmt.Errorf("runs-on cannot be empty string")
		}
	case []string:
		if len(v) == 0 {
			return nil, fmt.Errorf("runs-on cannot be empty slice")
		}
	case RunsOnGroupLabel:
		if v.Group == "" && v.Label == "" {
			return nil, fmt.Errorf("runs-on needs to be set with at least group or label")
		}
	default:
		return nil, fmt.Errorf("runs-on must be a string, slice of strings, or type with group/label, got: %T", r.value)
	}

	return r.value, nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (r *RunsOn) UnmarshalYAML(unmarshal func(any) error) error {
	// Try string first
	var str string
	if err := unmarshal(&str); err == nil {
		r.value = str

		return nil
	}

	// Try slice of strings
	var slice []string
	if err := unmarshal(&slice); err == nil {
		r.value = slice

		return nil
	}

	// Try group/label map
	var groupLabel RunsOnGroupLabel
	if err := unmarshal(&groupLabel); err == nil {
		r.value = groupLabel

		return nil
	}

	return fmt.Errorf("runs-on must be a string, slice of strings, or map with group/label, got: %T", r.value)
}

// NewRunsOnString creates a RunsOn from a string.
func NewRunsOnString(runner string) RunsOn {
	return RunsOn{value: runner}
}

// NewRunsOnSlice creates a RunsOn from a slice of strings.
func NewRunsOnSlice(runners []string) RunsOn {
	return RunsOn{value: runners}
}

// NewRunsOnGroupLabel creates a RunsOn from a group and label.
func NewRunsOnGroupLabel(group, label string) RunsOn {
	return RunsOn{value: RunsOnGroupLabel{Group: group, Label: label}}
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
	Uses            ActionRef         `yaml:"uses,omitempty"`
	With            map[string]string `yaml:"with,omitempty"`
	Env             map[string]string `yaml:"env,omitempty"`
	Run             string            `yaml:"run,omitempty"`
	ContinueOnError bool              `yaml:"continue-on-error,omitempty"`
	TimeoutMinutes  int               `yaml:"timeout-minutes,omitempty"`
}

type SlackNotifyPayload struct {
	Channel     string `json:"channel"`
	Text        string `json:"text"`
	IconEmoji   string `json:"icon_emoji"`
	Username    string `json:"username"`
	Attachments []any  `json:"attachments"`
}

// ActionRef represents a GitHub Action reference.
type ActionRef struct {
	Image   string
	Comment string
}

// MarshalYAML implements yaml.Marshaler.
func (a ActionRef) MarshalYAML() (any, error) {
	n := yaml.Node{}
	n.Kind = yaml.ScalarNode
	n.Tag = "!!str"
	n.Value = a.Image

	if a.Comment != "" {
		n.LineComment = a.Comment
	}

	return &n, nil
}

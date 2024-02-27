// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package ghactions provides GitHub Actions configuration.
package ghactions

import (
	"fmt"

	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Config provides GitHub Actions configuration.
type Config struct {
	dag.BaseNode

	meta *meta.Options

	// PullRequestEventType is the GitHub action event type to be used for the pull requests.
	//
	// Valid values are "pull_request" and "pull_request_target".
	PullRequestEventType string `yaml:"pullRequestEventType"`

	// CheckoutForkOnPullRequestTarget is a flag to checkout the fork repository and ref when "pull_request_target" event is used.
	//
	// This is a safety measure to avoid running the workflow on the forked repository by default.
	// Used only when pullRequestEventType is "pull_request_target".
	CheckoutForkOnPullRequestTarget bool `yaml:"checkoutForkOnPullRequestTarget"`
}

// NewConfig initializes Config.
func NewConfig(meta *meta.Options) *Config {
	return &Config{
		BaseNode: dag.NewBaseNode("ghactions-config"),

		meta: meta,

		PullRequestEventType: "pull_request",
	}
}

// CompileGitHubWorkflow implements ghworkflow.Compiler.
func (config *Config) CompileGitHubWorkflow(output *ghworkflow.Output) error {
	isPullRequestTarget := config.PullRequestEventType == "pull_request_target"

	if config.PullRequestEventType != "pull_request" && !isPullRequestTarget {
		return fmt.Errorf("unknown pull request event type: %s", config.PullRequestEventType)
	}

	output.UsePullRequestTargetEventType = isPullRequestTarget
	output.CheckoutForkOnPullRequestTarget = config.CheckoutForkOnPullRequestTarget

	return nil
}

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/output/sops"
	"github.com/siderolabs/kres/internal/project/meta"
)

// SOPS is a node that represents the sops configuration.
type SOPS struct {
	dag.BaseNode

	meta *meta.Options

	Config  string `yaml:"config"`
	Enabled bool   `yaml:"enabled"`
}

// NewSOPS creates a new SOPS node.
func NewSOPS(meta *meta.Options) *SOPS {
	return &SOPS{
		BaseNode: dag.NewBaseNode("sops"),

		meta: meta,

		Enabled: false,
	}
}

// CompileSops implements sops.Compiler.
func (sops *SOPS) CompileSops(o *sops.Output) error {
	if !sops.Enabled {
		return nil
	}

	sops.meta.SOPSEnabled = true

	o.Enable()
	o.Config(sops.Config)

	return nil
}

// CompileGitHubWorkflow implements ghworkflow.Compiler.
func (sops *SOPS) CompileGitHubWorkflow(o *ghworkflow.Output) error {
	// If sops is disabled or we are only compiling github workflows (since sops can be optionally enabled there), return early.
	if !sops.Enabled || sops.meta.CompileGithubWorkflowsOnly {
		return nil
	}

	o.AddStepAfter(
		ghworkflow.DefaultJobName,
		"setup-buildx",
		ghworkflow.SOPSSteps()...,
	)

	return nil
}

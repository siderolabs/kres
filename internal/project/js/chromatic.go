// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package js

import (
	"fmt"
	"strings"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Chromatic generates a standalone GitHub Actions workflow that publishes
// Storybook snapshots to Chromatic on every push and pull request.
type Chromatic struct {
	dag.BaseNode

	meta *meta.Options

	SOPSExtractKey string `yaml:"sopsExtractKey,omitempty"`
	Enabled        bool   `yaml:"enabled"`
}

// NewChromatic creates a new Chromatic node.
func NewChromatic(meta *meta.Options) *Chromatic {
	return &Chromatic{
		BaseNode: dag.NewBaseNode("chromatic"),
		meta:     meta,
	}
}

// CompileGitHubWorkflow implements ghworkflow.Compiler.
func (c *Chromatic) CompileGitHubWorkflow(o *ghworkflow.Output) error {
	if !c.Enabled {
		return nil
	}

	nodeVersion := strings.TrimSuffix(config.NodeContainerImageVersion, "-alpine")

	checkoutStep := ghworkflow.Step("Checkout code").
		SetUsesWithComment(
			"actions/checkout@"+config.CheckOutActionRef,
			"version: "+config.CheckOutActionVersion,
		).
		SetWith("fetch-depth", "0")

	setupNodeStep := ghworkflow.Step("Setup Node.js").
		SetUsesWithComment(
			"actions/setup-node@"+config.SetupNodeActionRef,
			"version: "+config.SetupNodeActionVersion,
		).
		SetWith("node-version", nodeVersion)

	installStep := &ghworkflow.JobStep{
		Name:             "Install dependencies",
		Run:              "npm ci\n",
		WorkingDirectory: "frontend/",
	}

	var getTokenStep *ghworkflow.JobStep

	if c.SOPSExtractKey != "" {
		getTokenStep = &ghworkflow.JobStep{
			Name: "Get token",
			Run: fmt.Sprintf(
				"chromaticProjectToken=$(sops decrypt --extract='%s' .secrets.yaml)\n"+
					"echo \"::add-mask::${chromaticProjectToken}\"\n"+
					"echo \"CHROMATIC_PROJECT_TOKEN=${chromaticProjectToken}\" >> $GITHUB_ENV\n",
				c.SOPSExtractKey,
			),
		}
	}

	chromaticStep := ghworkflow.Step("Run Chromatic").
		SetID("chromatic").
		SetUsesWithComment(
			"chromaui/action@"+config.ChromaticActionRef,
			"version: "+config.ChromaticActionVersion,
		).
		SetWith("projectToken", "${{ env.CHROMATIC_PROJECT_TOKEN }}").
		SetWith("workingDir", "frontend/")

	steps := []*ghworkflow.JobStep{checkoutStep, setupNodeStep, installStep}
	if getTokenStep != nil {
		steps = append(steps, getTokenStep)
	}

	steps = append(steps, chromaticStep)

	o.AddWorkflow(
		"chromatic",
		&ghworkflow.Workflow{
			Name: "chromatic",
			Concurrency: ghworkflow.Concurrency{
				Group:            "${{ github.workflow }}-${{ github.head_ref || github.run_id }}",
				CancelInProgress: true,
			},
			On: ghworkflow.On{
				Push: ghworkflow.Push{
					Branches: ghworkflow.Branches{
						c.meta.MainBranch,
						"release-*",
					},
					Tags: []string{"v*"},
				},
				PullRequest: ghworkflow.PullRequest{
					Branches: ghworkflow.Branches{
						c.meta.MainBranch,
						"release-*",
					},
				},
			},
			// https://www.chromatic.com/docs/github-actions/
			Jobs: map[string]*ghworkflow.Job{
				"chromatic": {
					Name:   "Run chromatic",
					RunsOn: ghworkflow.NewRunsOnGroupLabel(ghworkflow.GenericRunner, ""),
					Environment: &ghworkflow.JobEnvironment{
						Name: "Storybook preview",
						URL:  "${{ steps.chromatic.outputs.storybookUrl }}",
					},
					Steps: steps,
				},
			},
		},
	)

	return nil
}

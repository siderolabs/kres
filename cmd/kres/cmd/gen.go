// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/output"
	"github.com/siderolabs/kres/internal/output/codecov"
	"github.com/siderolabs/kres/internal/output/conform"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerignore"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/output/gitattributes"
	"github.com/siderolabs/kres/internal/output/github"
	"github.com/siderolabs/kres/internal/output/gitignore"
	"github.com/siderolabs/kres/internal/output/golangci"
	"github.com/siderolabs/kres/internal/output/lefthook"
	"github.com/siderolabs/kres/internal/output/license"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/output/markdownlint"
	"github.com/siderolabs/kres/internal/output/release"
	"github.com/siderolabs/kres/internal/output/renovate"
	"github.com/siderolabs/kres/internal/output/sops"
	"github.com/siderolabs/kres/internal/output/template"
	"github.com/siderolabs/kres/internal/project/auto"
	"github.com/siderolabs/kres/internal/project/meta"
)

var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate build instructions for the project.",
	Long: `Usage: kres gen

	Generate build instructions for the project. Kres analyzes the project structure
	starting from the current directory, detects project type and components, and emits
	build instructions in the following formats:

	  * Makefile
	  * Dockerfile`,
	Args: cobra.NoArgs,
	RunE: func(*cobra.Command, []string) error {
		return runGen()
	},
}

func runGen() error {
	fmt.Println("gen started")

	var err error

	options := meta.Options{
		GoContainerVersion:     fmt.Sprintf("%s-alpine", config.GoVersion),
		ContainerImageFrontend: config.ContainerImageFrontendDockerfile,
	}

	options.Config, err = config.NewProvider(".kres.yaml")
	if err != nil {
		return err
	}

	proj, err := auto.Build(&options)
	if err != nil {
		return err
	}

	if err := proj.LoadConfig(options.Config); err != nil {
		return err
	}

	gitattributesOutput := gitattributes.NewOutput()

	outputs := []output.Writer{
		output.Wrap(gitattributes.Manage(gitattributesOutput, github.NewOutput())),
		output.Wrap(gitattributes.Manage(gitattributesOutput, sops.NewOutput())),
		output.Wrap(gitattributes.Manage(gitattributesOutput, renovate.NewOutput())),
		output.Wrap(gitattributes.Manage(gitattributesOutput, conform.NewOutput())),
	}

	if !options.CompileGithubWorkflowsOnly {
		outputs = append(
			outputs,
			output.Wrap(gitattributes.Manage(gitattributesOutput, dockerfile.NewOutput())),
			output.Wrap(gitattributes.Manage(gitattributesOutput, dockerignore.NewOutput())),
			output.Wrap(gitattributes.Manage(gitattributesOutput, makefile.NewOutput())),
			output.Wrap(gitattributes.Manage(gitattributesOutput, golangci.NewOutput())),
			output.Wrap(gitattributes.Manage(gitattributesOutput, license.NewOutput())),
			output.Wrap(gitattributes.Manage(gitattributesOutput, gitignore.NewOutput())),
			output.Wrap(gitattributes.Manage(gitattributesOutput, codecov.NewOutput())),
			output.Wrap(gitattributes.Manage(gitattributesOutput, release.NewOutput())),
			output.Wrap(gitattributes.Manage(gitattributesOutput, markdownlint.NewOutput())),
			output.Wrap(gitattributes.Manage(gitattributesOutput, template.NewOutput())),
			output.Wrap(gitattributes.Manage(gitattributesOutput, lefthook.NewOutput())),
		)
	}

	outputs = append(outputs, output.Wrap(gitattributes.Manage(gitattributesOutput, ghworkflow.NewOutput(
		options.MainBranch,
		!options.CompileGithubWorkflowsOnly,
		!options.SkipStaleWorkflow,
		options.CIFailureSlackNotifyChannel,
	))))
	outputs = append(outputs, output.Wrap(gitattributes.Manage(gitattributesOutput, gitattributesOutput)))

	if err := proj.Compile(outputs); err != nil {
		return err
	}

	for _, out := range outputs {
		if err := out.Generate(); err != nil {
			return fmt.Errorf("failed on step '%T', error: %w", out, err)
		}
	}

	fmt.Println("success")

	return nil
}

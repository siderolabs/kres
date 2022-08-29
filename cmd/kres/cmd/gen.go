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
	"github.com/siderolabs/kres/internal/output/drone"
	"github.com/siderolabs/kres/internal/output/github"
	"github.com/siderolabs/kres/internal/output/gitignore"
	"github.com/siderolabs/kres/internal/output/golangci"
	"github.com/siderolabs/kres/internal/output/license"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/output/markdownlint"
	"github.com/siderolabs/kres/internal/output/release"
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
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runGen()
	},
}

func runGen() error {
	fmt.Println("gen started")

	outputs := []output.Writer{
		dockerfile.NewOutput(),
		makefile.NewOutput(),
		golangci.NewOutput(),
		license.NewOutput(),
		gitignore.NewOutput(),
		drone.NewOutput(),
		codecov.NewOutput(),
		release.NewOutput(),
		markdownlint.NewOutput(),
		github.NewOutput(),
		conform.NewOutput(),
		template.NewOutput(),
	}

	var err error

	options := meta.Options{
		GoContainerVersion: config.GolangContainerImageVersion,
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

	if err := proj.Compile(outputs); err != nil {
		return err
	}

	for _, out := range outputs {
		if err := out.Generate(); err != nil {
			return err
		}
	}

	fmt.Println("success")

	return nil
}

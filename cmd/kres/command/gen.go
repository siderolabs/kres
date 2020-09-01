// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package command

import (
	"strings"

	"github.com/mitchellh/cli"

	"github.com/talos-systems/kres/internal/config"
	"github.com/talos-systems/kres/internal/output"
	"github.com/talos-systems/kres/internal/output/codecov"
	"github.com/talos-systems/kres/internal/output/conform"
	"github.com/talos-systems/kres/internal/output/dockerfile"
	"github.com/talos-systems/kres/internal/output/drone"
	"github.com/talos-systems/kres/internal/output/github"
	"github.com/talos-systems/kres/internal/output/gitignore"
	"github.com/talos-systems/kres/internal/output/golangci"
	"github.com/talos-systems/kres/internal/output/license"
	"github.com/talos-systems/kres/internal/output/makefile"
	"github.com/talos-systems/kres/internal/output/markdownlint"
	"github.com/talos-systems/kres/internal/output/release"
	"github.com/talos-systems/kres/internal/project/auto"
	"github.com/talos-systems/kres/internal/project/meta"
)

// Gen implements 'gen' command.
type Gen struct {
	Meta
}

// Help implements cli.Command.
func (c *Gen) Help() string {
	helpText := `
Usage: kres gen

	Generate build instructions for the project. Kres analyzes the project structure
	starting from the current directory, detects project type and components, and emits
	build instructions in the following formats:

	  * Makefile
	  * Dockerfile

Options:

	--outputs=output1,output2           Additional outputs to be generated
`

	return strings.TrimSpace(helpText)
}

// Synopsis implements cli.Command.
func (c *Gen) Synopsis() string {
	return "Generate build instructions for the project."
}

// Run implements cli.Command.
func (c *Gen) Run(args []string) int {
	c.Ui.Info("gen started")

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
	}

	var err error

	options := meta.Options{}

	options.Config, err = config.NewProvider(".kres.yaml")
	if err != nil {
		c.Ui.Error(err.Error())

		return 1
	}

	proj, err := auto.Build(&options)
	if err != nil {
		c.Ui.Error(err.Error())

		return 1
	}

	if err := proj.LoadConfig(options.Config); err != nil {
		c.Ui.Error(err.Error())

		return 1
	}

	if err := proj.Compile(outputs); err != nil {
		c.Ui.Error(err.Error())

		return 1
	}

	for _, out := range outputs {
		if err := out.Generate(); err != nil {
			c.Ui.Error(err.Error())

			return 1
		}
	}

	c.Ui.Info("success")

	return 0
}

// NewGen creates Gen command.
func NewGen(m Meta) cli.CommandFactory {
	return func() (cli.Command, error) {
		return &Gen{
			Meta: m,
		}, nil
	}
}

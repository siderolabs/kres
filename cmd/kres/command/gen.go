// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package command

import (
	"strings"

	"github.com/mitchellh/cli"

	"github.com/talos-systems/kres/internal/output"
	"github.com/talos-systems/kres/internal/output/dockerfile"
	"github.com/talos-systems/kres/internal/output/drone"
	"github.com/talos-systems/kres/internal/output/gitignore"
	"github.com/talos-systems/kres/internal/output/golangci"
	"github.com/talos-systems/kres/internal/output/license"
	"github.com/talos-systems/kres/internal/output/makefile"
	"github.com/talos-systems/kres/internal/project"
	"github.com/talos-systems/kres/internal/project/auto"
	"github.com/talos-systems/kres/internal/project/common"
	"github.com/talos-systems/kres/internal/project/golang"
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
	}

	options := meta.Options{}

	golangDetected, err := auto.DetectGolang(".", &options)
	if err != nil {
		c.Ui.Error(err.Error())

		return 1
	}

	if golangDetected {
		c.Ui.Info("Go project detected")
	}

	proj := &project.Contents{}

	commonBuild := common.NewBuild(&options)
	commonDocker := common.NewDocker(&options)

	golangciLint := golang.NewGolangciLint(&options)
	gofumpt := golang.NewGofumpt(&options)

	toolchain := golang.NewToolchain(&options)
	toolchain.AddInput(commonBuild, commonDocker, golangciLint, gofumpt)

	build := golang.NewBuild(&options, "kres", "cmd/kres")
	build.AddInput(toolchain)

	unitTests := golang.NewUnitTests(&options)
	unitTests.AddInput(toolchain)

	lint := common.NewLint(&options)
	lint.AddInput(toolchain, golangciLint, gofumpt)

	image := common.NewImage(&options, "kres")
	image.AddInput(build, common.NewFHS(&options), common.NewCACerts(&options), lint)

	proj.AddTarget(build, lint, unitTests, image)

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

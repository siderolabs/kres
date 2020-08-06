// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"os"

	"github.com/mitchellh/cli"

	"github.com/talos-systems/kres/cmd/kres/command"
	"github.com/talos-systems/kres/internal/version"
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	ui := &cli.ColoredUi{
		InfoColor:  cli.UiColorCyan,
		WarnColor:  cli.UiColorYellow,
		ErrorColor: cli.UiColorRed,
		Ui: &cli.BasicUi{
			Reader:      os.Stdin,
			Writer:      os.Stdout,
			ErrorWriter: os.Stderr,
		},
	}

	meta := command.Meta{
		Ui: ui,
	}

	c := cli.NewCLI(version.Name, version.Tag)
	c.Args = os.Args[1:]
	c.Commands = map[string]cli.CommandFactory{
		"gen":     command.NewGen(meta),
		"version": command.NewVersion(meta),
	}
	c.HelpWriter = os.Stdout
	c.ErrorWriter = os.Stderr

	exitStatus, err := c.Run()
	if err != nil {
		ui.Error(err.Error())
	}

	return exitStatus
}

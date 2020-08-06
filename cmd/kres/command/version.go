// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package command

import (
	"fmt"

	"github.com/mitchellh/cli"

	"github.com/talos-systems/kres/internal/version"
)

// Version implements 'version' command.
type Version struct {
	Meta
}

// Help implements cli.Command.
func (c *Version) Help() string {
	return "Prints Kres version."
}

// Synopsis implements cli.Command.
func (c *Version) Synopsis() string {
	return c.Help()
}

// Run implements cli.Command.
func (c *Version) Run(args []string) int {
	c.Ui.Output(fmt.Sprintf("%s version %s (%s)", version.Name, version.Tag, version.SHA))

	return 0
}

// NewVersion creates Version command.
func NewVersion(m Meta) cli.CommandFactory {
	return func() (cli.Command, error) {
		return &Version{
			Meta: m,
		}, nil
	}
}

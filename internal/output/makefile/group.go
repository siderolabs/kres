// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package makefile

import (
	"fmt"
	"io"
)

// Descriptions (used as keys) of some predefined variable groups.
const (
	VariableGroupCommon = "common variables"
	VariableGroupDocker = "docker build settings"
	VariableGroupHelp   = "help menu"
	VariableGroupExtra  = "extra variables"
)

// VariableGroup is a way to group nicely variables in Makefile.
type VariableGroup struct {
	variables   []*Variable
	description string
}

// Variable appends variable to the group.
func (group *VariableGroup) Variable(variable *Variable) *VariableGroup {
	group.variables = append(group.variables, variable)

	return group
}

// Generate renders group to output.
func (group *VariableGroup) Generate(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "# %s\n\n", group.description); err != nil {
		return err
	}

	for _, variable := range group.variables {
		if err := variable.Generate(w); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintln(w)

	return err
}

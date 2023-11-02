// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package makefile

import (
	"fmt"
	"io"
	"slices"
)

// Descriptions (used as keys) of some predefined variable groups.
const (
	VariableGroupCommon            = "common variables"
	VariableGroupDocker            = "docker build settings"
	VariableGroupExtra             = "extra variables"
	VariableGroupHelp              = "help menu"
	VariableGroupSourceDateEpoch   = "source date epoch of first commit"
	VariableGroupTargets           = "targets defines all the available targets"
	VariableGroupAdditionalTargets = "additional targets defines all the additional targets"
)

// VariableGroup is a way to group nicely variables in Makefile.
type VariableGroup struct {
	description string
	variables   []*Variable
}

// Variable appends variable to the group.
func (group *VariableGroup) Variable(variable *Variable) *VariableGroup {
	if slices.ContainsFunc(group.variables, func(item *Variable) bool {
		return item.name == variable.name
	}) {
		return group
	}

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

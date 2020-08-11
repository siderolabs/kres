// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package makefile

import (
	"fmt"
	"io"
	"strings"
)

// Variable abstract Makefile variable of different flavors.
type Variable struct {
	name     string
	operator string
	value    string
}

// RecursiveVariable creates new Variable of recursive (=) flavor.
func RecursiveVariable(name, value string) *Variable {
	return &Variable{
		name:     name,
		operator: "=",
		value:    value,
	}
}

// OverridableVariable creates new overridable Variable of recursive (?=) flavor.
func OverridableVariable(name, value string) *Variable {
	return &Variable{
		name:     name,
		operator: "?=",
		value:    value,
	}
}

// SimpleVariable creates new Variable with simple evaluation (:=).
func SimpleVariable(name, value string) *Variable {
	return &Variable{
		name:     name,
		operator: ":=",
		value:    value,
	}
}

// Push is used to push extra value (+=) to RecursiveVariable.
func (variable *Variable) Push(line string) *Variable {
	variable.value += "\n" + line

	return variable
}

// Generate renders variable definition.
func (variable *Variable) Generate(w io.Writer) error {
	if variable.operator == "=" && strings.ContainsRune(variable.value, '\n') {
		lines := strings.Split(variable.value, "\n")

		for i, line := range lines {
			operator := "+="
			if i == 0 {
				operator = variable.operator
			}

			_, err := fmt.Fprintf(w, "%s %s %s\n", variable.name, operator, line)
			if err != nil {
				return err
			}
		}

		return nil
	}

	_, err := fmt.Fprintf(w, "%s %s %s\n", variable.name, variable.operator, variable.value)

	return err
}

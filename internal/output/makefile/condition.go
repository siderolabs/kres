// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package makefile

import (
	"fmt"
	"io"
)

// Condition is a if-clause in Makefile.
type Condition struct {
	trigger    *Trigger
	thenClause []*Variable
	elseClause []*Variable
}

// Trigger is the expression for the Condition.
type Trigger struct {
	// only support "true" check for now
	variable string
}

// Then adds a variable to the then-clause.
func (condition *Condition) Then(vars ...*Variable) *Condition {
	condition.thenClause = append(condition.thenClause, vars...)

	return condition
}

// Else adds a variable to the else-clause.
func (condition *Condition) Else(vars ...*Variable) *Condition {
	condition.elseClause = append(condition.elseClause, vars...)

	return condition
}

// Generate output for the Makefile.
func (condition *Condition) Generate(w io.Writer) error {
	// only support "true" check for now
	if _, err := fmt.Fprintf(w, "ifneq (, $(filter $(%s), t true TRUE y yes 1))\n", condition.trigger.variable); err != nil {
		return err
	}

	for _, variable := range condition.thenClause {
		if err := variable.Generate(w); err != nil {
			return err
		}
	}

	if len(condition.elseClause) > 0 {
		if _, err := fmt.Fprint(w, "else\n"); err != nil {
			return err
		}

		for _, variable := range condition.elseClause {
			if err := variable.Generate(w); err != nil {
				return err
			}
		}
	}

	if _, err := fmt.Fprint(w, "endif\n\n"); err != nil {
		return err
	}

	return nil
}

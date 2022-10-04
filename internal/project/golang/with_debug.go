// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang

import (
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// WithDebugVarName is the variable name for enabling debug mode by adding the special build tag "sidero.debug".
const WithDebugVarName = "WITH_DEBUG"

// WithDebug is the node which adds the special build tag "sidero.debug" to the build.
type WithDebug struct {
	dag.BaseNode
}

// NewWithDebug creates a new WithDebug node.
func NewWithDebug(meta *meta.Options) *WithDebug {
	meta.BuildArgs = append(meta.BuildArgs, WithDebugVarName)

	return &WithDebug{}
}

// CompileMakefile implements makefile.Compiler.
func (build *WithDebug) CompileMakefile(output *makefile.Output) error {
	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable(WithDebugVarName, "false"))

	return nil
}

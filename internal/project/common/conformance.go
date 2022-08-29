// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Conformance builds Makefile `conformance` target.
type Conformance struct {
	dag.BaseNode

	meta *meta.Options

	ConformanceImage string `yaml:"conformanceImage"`
}

// NewConformance initializes Conformance.
func NewConformance(meta *meta.Options) *Conformance {
	return &Conformance{
		BaseNode: dag.NewBaseNode("conformance"),

		meta: meta,

		ConformanceImage: "ghcr.io/siderolabs/conform:latest",
	}
}

const conformanceImageEnvVarName = "CONFORMANCE_IMAGE"

// CompileMakefile implements makefile.Compiler.
func (conformance *Conformance) CompileMakefile(output *makefile.Output) error {
	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable(conformanceImageEnvVarName, conformance.ConformanceImage))

	output.Target(conformance.Name()).
		Script("@docker pull $(" + conformanceImageEnvVarName + ")").
		Script("@docker run --rm -it -v $(PWD):/src -w /src $(" + conformanceImageEnvVarName + ") enforce").
		Phony()

	return nil
}

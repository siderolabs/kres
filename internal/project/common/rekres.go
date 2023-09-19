// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// ReKres builds Makefile `rekres` target.
type ReKres struct {
	dag.BaseNode

	meta *meta.Options

	KresImage string `yaml:"kresImage"`
}

// NewReKres initializes ReKres.
func NewReKres(meta *meta.Options) *ReKres {
	return &ReKres{
		BaseNode: dag.NewBaseNode("rekres"),

		meta: meta,

		KresImage: "ghcr.io/siderolabs/kres:latest",
	}
}

// CompileMakefile implements makefile.Compiler.
func (rekres *ReKres) CompileMakefile(output *makefile.Output) error {
	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable("KRES_IMAGE", rekres.KresImage))

	output.Target(rekres.Name()).
		Script("@docker pull $(KRES_IMAGE)").
		Script("@docker run --rm --net=host -v $(PWD):/src -w /src -e GITHUB_TOKEN $(KRES_IMAGE)").
		Phony()

	return nil
}

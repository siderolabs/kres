// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"github.com/talos-systems/kres/internal/dag"
	"github.com/talos-systems/kres/internal/output/makefile"
	"github.com/talos-systems/kres/internal/project/meta"
)

// All builds Makefile `all` target.
type All struct {
	dag.BaseNode

	meta *meta.Options
}

// NewAll initializes All.
func NewAll(meta *meta.Options) *All {
	return &All{
		BaseNode: dag.NewBaseNode("all"),

		meta: meta,
	}
}

// CompileMakefile implements makefile.Compiler.
func (all *All) CompileMakefile(output *makefile.Output) error {
	output.Target("all").
		Depends(all.InputNames()...)

	return nil
}

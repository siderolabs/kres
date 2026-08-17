// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/gitattributes"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Gitattributes is a node that represents the .gitattributes configuration.
type Gitattributes struct {
	dag.BaseNode

	meta *meta.Options

	AdditionalContent []string `yaml:"additionalContent"`
}

// NewGitattributes creates a new Gitattributes node.
func NewGitattributes(meta *meta.Options) *Gitattributes {
	return &Gitattributes{
		BaseNode: dag.NewBaseNode("gitattributes"),

		meta: meta,
	}
}

// CompileGitattributes implements gitattributes.Compiler.
func (g *Gitattributes) CompileGitattributes(o *gitattributes.Output) error {
	o.AddContent(g.AdditionalContent...)

	return nil
}

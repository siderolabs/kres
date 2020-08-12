// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package auto provides automatic detector of project type, reflections.
package auto

import (
	"github.com/talos-systems/kres/internal/project"
	"github.com/talos-systems/kres/internal/project/meta"
)

// Build the project type and structure based on project type.
func Build(meta *meta.Options) (*project.Contents, error) {
	builder := newBuilder(meta)

	if err := builder.build(); err != nil {
		return nil, err
	}

	return builder.proj, nil
}

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package js_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/drone"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/output/template"
	"github.com/siderolabs/kres/internal/project/js"
)

func TestToolchainInterfaces(t *testing.T) {
	assert.Implements(t, (*dockerfile.Compiler)(nil), new(js.Toolchain))
	assert.Implements(t, (*template.Compiler)(nil), new(js.Toolchain))
	assert.Implements(t, (*makefile.Compiler)(nil), new(js.Toolchain))
	assert.Implements(t, (*drone.Compiler)(nil), new(js.Toolchain))
	assert.Implements(t, (*makefile.SkipAsMakefileDependency)(nil), new(js.Toolchain))
}

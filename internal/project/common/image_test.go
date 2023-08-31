// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/drone"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/common"
)

func TestImageInterfaces(t *testing.T) {
	assert.Implements(t, (*makefile.Compiler)(nil), new(common.Image))
	assert.Implements(t, (*dockerfile.Compiler)(nil), new(common.Image))
	assert.Implements(t, (*drone.Compiler)(nil), new(common.Image))
	assert.Implements(t, (*ghworkflow.Compiler)(nil), new(common.Image))
}

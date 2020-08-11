// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/talos-systems/kres/internal/output/dockerfile"
	"github.com/talos-systems/kres/internal/project/common"
)

func TestInputImageInterfaces(t *testing.T) {
	assert.Implements(t, (*dockerfile.Compiler)(nil), new(common.InputImage))
}

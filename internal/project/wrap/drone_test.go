// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package wrap_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/siderolabs/kres/internal/output/drone"
	"github.com/siderolabs/kres/internal/project/wrap"
)

func TestDroneInterfaces(t *testing.T) {
	assert.Implements(t, (*drone.Compiler)(nil), wrap.Drone(nil))
}

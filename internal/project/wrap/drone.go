// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package wrap

import (
	"github.com/talos-systems/kres/internal/dag"
	"github.com/talos-systems/kres/internal/output/drone"
)

// DroneWrapper wraps the node so that it has only drone.Compiler interface exposed.
type DroneWrapper struct {
	dag.Node
}

// Drone returns new DroneWrapper.
func Drone(wrapped dag.Node) *DroneWrapper {
	return &DroneWrapper{wrapped}
}

// CompileDrone implements drone.Compiler interface.
func (drone *DroneWrapper) CompileDrone(*drone.Output) error {
	return nil
}

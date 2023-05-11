// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package drone

import (
	"github.com/drone/drone-yaml/yaml"
	"github.com/siderolabs/gen/slices"
)

// Service appends a new service.
func (o *Output) Service(spec *yaml.Container) {
	o.appendService(spec, o.defaultPipeline)
}

func (o *Output) appendService(originalService *yaml.Container, pipeline *yaml.Pipeline) {
	// perform a shallow copy of the step to avoid modifying the original
	spec := *originalService

	spec.Volumes = slices.Clone(spec.Volumes)
	spec.Volumes = append(spec.Volumes, o.standardMounts...)

	pipeline.Services = append(pipeline.Services, &spec)
}

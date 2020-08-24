// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package drone

import "github.com/drone/drone-yaml/yaml"

// Service appends a new service.
func (o *Output) Service(spec *yaml.Container) *Output {
	spec.Volumes = append(spec.Volumes, o.standardMounts...)

	o.defaultPipeline.Services = append(o.defaultPipeline.Services, spec)

	return o
}

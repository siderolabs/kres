// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package drone

import "github.com/drone/drone-yaml/yaml"

// Pipeline wraps custom pipelines to handle step appending.
type Pipeline struct {
	drone     *Output
	pipelines []*yaml.Pipeline
}

// Step appends a step to the pipeline.
func (p *Pipeline) Step(step *Step) {
	for _, pipeline := range p.pipelines {
		p.drone.appendStep(step, pipeline)
	}
}

// Service appends a new service.
func (p *Pipeline) Service(service *yaml.Container) {
	for _, pipeline := range p.pipelines {
		p.drone.appendService(service, pipeline)
	}
}

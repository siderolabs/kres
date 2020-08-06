// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package drone

import (
	"fmt"

	"github.com/drone/drone-yaml/yaml"
)

// Step is a pipeline Step.
type Step struct {
	container yaml.Container
}

// MakeStep creates a step which calls make target.
func MakeStep(target string) *Step {
	return &Step{
		container: yaml.Container{
			Name: target,
			Commands: []string{
				fmt.Sprintf("make %s", target),
			},
			Environment: make(map[string]*yaml.Variable),
		},
	}
}

// Name provides a name to a step.
func (step *Step) Name(name string) *Step {
	step.container.Name = name

	return step
}

// Environment appends an environment variable to the step.
func (step *Step) Environment(name, value string) *Step {
	step.container.Environment[name] = &yaml.Variable{Value: value}

	return step
}

// DependsOn appends to a list of step dependencies.
func (step *Step) DependsOn(depends ...string) *Step {
	step.container.DependsOn = append(step.container.DependsOn, depends...)

	return step
}

// ExceptPullRequest adds condition to skip step on PRs.
func (step *Step) ExceptPullRequest() *Step {
	step.container.When.Event.Exclude = []string{"pull_request"}

	return step
}

// LocalRegistry sets up pushing to local registry.
func (step *Step) LocalRegistry() *Step {
	step.container.Environment["REGISTRY"] = &yaml.Variable{
		Value: "registry.ci.svc:5000",
	}

	return step
}

// DockerLogin sets up login to registry.
func (step *Step) DockerLogin() *Step {
	step.container.Commands = append([]string{
		`docker login --username "$${DOCKER_USERNAME}" --password "$${DOCKER_PASSWORD}"`,
	}, step.container.Commands...)

	step.container.Environment["DOCKER_USERNAME"] = &yaml.Variable{
		Secret: "docker_username",
	}
	step.container.Environment["DOCKER_PASSWORD"] = &yaml.Variable{
		Secret: "docker_password",
	}

	return step
}

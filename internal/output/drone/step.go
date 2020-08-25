// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package drone

import (
	"fmt"
	"strings"

	"github.com/drone/drone-yaml/yaml"
)

// Step is a pipeline Step.
type Step struct {
	container yaml.Container
}

// MakeStep creates a step which calls make target.
func MakeStep(target string, args ...string) *Step {
	return &Step{
		container: yaml.Container{
			Name: target,
			Commands: []string{
				strings.TrimSpace(fmt.Sprintf("make %s %s", target, strings.Join(args, " "))),
			},
			Environment: make(map[string]*yaml.Variable),
		},
	}
}

// CustomStep creates a step which calls some shell script.
func CustomStep(target string, commands ...string) *Step {
	return &Step{
		container: yaml.Container{
			Name:        target,
			Commands:    commands,
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

// EnvironmentFromSecret appends an environment variable from secret to the step.
func (step *Step) EnvironmentFromSecret(name, secretName string) *Step {
	step.container.Environment[name] = &yaml.Variable{Secret: secretName}

	return step
}

// DependsOn appends to a list of step dependencies.
func (step *Step) DependsOn(depends ...string) *Step {
	step.container.DependsOn = append(step.container.DependsOn, depends...)

	return step
}

// ExceptPullRequest adds condition to skip step on PRs.
func (step *Step) ExceptPullRequest() *Step {
	step.container.When.Event.Exclude = append(step.container.When.Event.Exclude, "pull_request")

	return step
}

// OnlyOnPullRequest adds condition to run step only on PRs.
func (step *Step) OnlyOnPullRequest() *Step {
	step.container.When.Event.Include = append(step.container.When.Event.Include, "pull_request")

	return step
}

// OnlyOnMaster adds condition to run step only on master branch.
func (step *Step) OnlyOnMaster() *Step {
	step.container.When.Branch.Include = append(step.container.When.Branch.Include, "master")

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

// Privileged marks step as privileged.
func (step *Step) Privileged() *Step {
	step.container.Privileged = true

	return step
}

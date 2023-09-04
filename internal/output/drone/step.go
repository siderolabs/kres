// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package drone

import (
	"fmt"
	"slices"
	"strings"

	"github.com/drone/drone-yaml/yaml"
)

// Step is a pipeline Step.
type Step struct {
	container yaml.Container
	volumes   []*yaml.Volume
}

// MakeStep creates a step which calls make target.
func MakeStep(target string, args ...string) *Step {
	return &Step{
		container: yaml.Container{
			Name: target,
			Commands: []string{
				strings.TrimSpace(fmt.Sprintf("make %s %s", target, strings.Join(args, " "))),
			},
			Environment: map[string]*yaml.Variable{},
		},
	}
}

// CustomStep creates a step which calls some shell script.
func CustomStep(target string, commands ...string) *Step {
	return &Step{
		container: yaml.Container{
			Name:        target,
			Commands:    commands,
			Environment: map[string]*yaml.Variable{},
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
	for _, dep := range depends {
		if slices.Contains(step.container.DependsOn, dep) {
			continue
		}

		step.container.DependsOn = append(step.container.DependsOn, dep)
	}

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

// OnlyOnTag adds condition to run step only on tags.
func (step *Step) OnlyOnTag() *Step {
	step.container.When.Event.Include = append(step.container.When.Event.Include, "tag")

	return step
}

// OnlyOnBranch adds condition to run step only on the specified branch.
func (step *Step) OnlyOnBranch(branchName string) *Step {
	step.container.When.Branch.Include = append(step.container.When.Branch.Include, branchName)

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
		`docker login ghcr.io --username "$${GHCR_USERNAME}" --password "$${GHCR_PASSWORD}"`,
	}, step.container.Commands...)

	step.container.Environment["GHCR_USERNAME"] = &yaml.Variable{
		Secret: "ghcr_username",
	}
	step.container.Environment["GHCR_PASSWORD"] = &yaml.Variable{
		Secret: "ghcr_token",
	}

	return step
}

// Privileged marks step as privileged.
func (step *Step) Privileged() *Step {
	step.container.Privileged = true

	return step
}

// Image sets step image.
func (step *Step) Image(image string) *Step {
	step.container.Image = image

	return step
}

// ResourceRequests sets step resource requests.
func (step *Step) ResourceRequests(cpuCores, memoryGib int) *Step {
	if step.container.Resources == nil {
		step.container.Resources = &yaml.Resources{}
	}

	if step.container.Resources.Requests == nil {
		step.container.Resources.Requests = &yaml.ResourceObject{}
	}

	step.container.Resources.Requests.CPU = float64(cpuCores) * 1000.0
	step.container.Resources.Requests.Memory = yaml.BytesSize(int64(memoryGib) * 1024 * 1024 * 1024)

	return step
}

// PublishArtifacts publishes artifacts with the default Github settings.
func (step *Step) PublishArtifacts(note string, artifacts ...string) *Step {
	step.container.Settings = map[string]*yaml.Parameter{
		"api_key": {
			Secret: "github_token",
		},
		"checksum": {
			Value: []string{"sha256", "sha512"},
		},
		"draft": {
			Value: true,
		},
		"files": {
			Value: artifacts,
		},
		"note": {
			Value: note,
		},
	}

	return step
}

// EmptyDirVolume mounts an empty dir volume to the step.
func (step *Step) EmptyDirVolume(name, mountPath string) *Step {
	step.container.Volumes = append(step.container.Volumes, &yaml.VolumeMount{
		Name:      name,
		MountPath: mountPath,
	})

	step.volumes = append(step.volumes, &yaml.Volume{
		Name:     name,
		EmptyDir: &yaml.VolumeEmptyDir{},
	})

	return step
}

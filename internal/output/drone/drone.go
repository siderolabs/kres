// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package drone implements output to .drone.yml.
package drone

import (
	"bytes"
	"io"

	"github.com/drone/drone-yaml/yaml"
	"github.com/drone/drone-yaml/yaml/pretty"

	"github.com/talos-systems/kres/internal/output"
)

const (
	filename = ".drone.yml"
)

// Output implements Drone project config generation.
type Output struct {
	output.FileAdapter

	manifest *yaml.Manifest

	defaultPipeline *yaml.Pipeline
	notifyPipeline  *yaml.Pipeline

	standardMounts []*yaml.VolumeMount

	PipelineType       string
	NotifySlackChannel string
	BuildContainer     string
	DockerImage        string
}

// NewOutput creates new .drone.yml output.
func NewOutput() *Output {
	output := &Output{
		manifest: &yaml.Manifest{},

		PipelineType:       "kubernetes",
		NotifySlackChannel: "proj-talos-maintainers",
		BuildContainer:     buildContainer,
		DockerImage:        "docker:19.03-dind",
	}

	output.standardMounts = []*yaml.VolumeMount{
		{
			Name:      "outer-docker-socket",
			MountPath: "/var/outer-run",
		},
		{
			Name:      "docker-socket",
			MountPath: "/var/run",
		},
		{
			Name:      "ssh",
			MountPath: "/root/.ssh",
		},
		{
			Name:      "buildx",
			MountPath: "/root/.docker/buildx",
		},
	}

	output.defaultPipeline = &yaml.Pipeline{
		Name: "default",
		Type: output.PipelineType,
		Kind: "pipeline",
		Volumes: []*yaml.Volume{
			{
				Name: "outer-docker-socket",
				HostPath: &yaml.VolumeHostPath{
					Path: "/var/ci-docker",
				},
			},
			{
				Name: "docker-socket",
				EmptyDir: &yaml.VolumeEmptyDir{
					Medium: "memory",
				},
			},
			{
				Name: "buildx",
				EmptyDir: &yaml.VolumeEmptyDir{
					Medium: "memory",
				},
			},
			{
				Name: "ssh",
				EmptyDir: &yaml.VolumeEmptyDir{
					Medium: "memory",
				},
			},
		},
		Steps: []*yaml.Container{
			{
				Name:  "setup-ci",
				Image: output.BuildContainer,
				Pull:  "always",
				Commands: []string{
					"sleep 5",
					"git fetch --tags",
					"install-ci-key",
					"docker buildx create --driver docker-container --platform linux/amd64 --name local --use unix:///var/outer-run/docker.sock",
					"docker buildx inspect --bootstrap",
				},
				Environment: map[string]*yaml.Variable{
					"SSH_KEY": {
						Secret: "ssh_key",
					},
				},
				Volumes: output.standardMounts,
			},
		},
		Services: []*yaml.Container{
			{
				Name:       "docker",
				Image:      output.DockerImage,
				Entrypoint: []string{"dockerd"},
				Privileged: true,
				Commands: []string{
					"--dns=8.8.8.8",
					"--dns=8.8.4.4",
					"--mtu=1500",
					"--log-level=error",
					"--insecure-registry=http://registry.ci.svc:5000",
				},
				Volumes: output.standardMounts,
			},
		},
	}

	output.notifyPipeline = &yaml.Pipeline{
		Name: "notify",
		Type: output.PipelineType,
		Kind: "pipeline",
		Clone: yaml.Clone{
			Disable: true,
		},
		Trigger: yaml.Conditions{
			Status: yaml.Condition{
				Include: []string{"success", "failure"},
			},
		},
		Steps: []*yaml.Container{
			{
				Name:  "slack",
				Image: "plugins/slack",
				Settings: map[string]*yaml.Parameter{
					"webhook": {
						Secret: "slack_webhook",
					},
					"channel": {
						Value: output.NotifySlackChannel,
					},
					"link_names": {
						Value: true,
					},
					"template": {
						Value: notifyTemplate,
					},
				},
				When: yaml.Conditions{
					Status: yaml.Condition{
						Include: []string{"success", "failure"},
					},
				},
			},
		},
		DependsOn: []string{"default"},
	}

	output.manifest.Resources = append(output.manifest.Resources, output.defaultPipeline, output.notifyPipeline)

	output.FileAdapter.FileWriter = output

	return output
}

// Step appends a step to the default pipeline.
func (o *Output) Step(step *Step) {
	if step.container.Image == "" {
		step.container.Image = o.BuildContainer
	}

	if step.container.Pull == "" {
		step.container.Pull = "always"
	}

	step.container.Volumes = append(step.container.Volumes, o.standardMounts...)

	o.defaultPipeline.Steps = append(o.defaultPipeline.Steps, &step.container)
}

// Compile implements output.Writer interface.
func (o *Output) Compile(node interface{}) error {
	compiler, implements := node.(Compiler)

	if !implements {
		return nil
	}

	return compiler.CompileDrone(o)
}

// Filenames implements output.FileWriter interface.
func (o *Output) Filenames() []string {
	return []string{filename}
}

// GenerateFile implements output.FileWriter interface.
func (o *Output) GenerateFile(filename string, w io.Writer) error {
	switch filename {
	case filename:
		return o.drone(w)
	default:
		panic("unexpected filename: " + filename)
	}
}

func (o *Output) drone(w io.Writer) error {
	preamble := output.Preamble("# ")

	var buf bytes.Buffer

	pretty.Print(&buf, o.manifest)

	firstLine, err := buf.ReadString('\n')
	if err != nil {
		return err
	}

	if _, err := w.Write([]byte(firstLine)); err != nil {
		return err
	}

	if _, err := w.Write([]byte(preamble)); err != nil {
		return err
	}

	if _, err := buf.WriteTo(w); err != nil {
		return err
	}

	return nil
}

// Compiler is implemented by project blocks which support Drone config generation.
type Compiler interface {
	CompileDrone(*Output) error
}

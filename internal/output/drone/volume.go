// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package drone

import "github.com/drone/drone-yaml/yaml"

// VolumeHostPath adds a host path mount.
func (o *Output) VolumeHostPath(name, hostPath, mountPath string) *Output {
	o.standardVolumes = append(o.standardVolumes, &yaml.Volume{
		Name: name,
		HostPath: &yaml.VolumeHostPath{
			Path: hostPath,
		},
	})

	o.standardMounts = append(o.standardMounts, &yaml.VolumeMount{
		Name:      name,
		MountPath: mountPath,
	})

	return o
}

// VolumeTemporary adds a temporary (tmpfs) volume mount.
func (o *Output) VolumeTemporary(name, mountPath string) *Output {
	o.standardVolumes = append(o.standardVolumes, &yaml.Volume{
		Name: name,
		EmptyDir: &yaml.VolumeEmptyDir{
			Medium: "memory",
		},
	})

	o.standardMounts = append(o.standardMounts, &yaml.VolumeMount{
		Name:      name,
		MountPath: mountPath,
	})

	return o
}

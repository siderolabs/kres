// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package auto

import (
	"github.com/siderolabs/gen/maps"

	"github.com/siderolabs/kres/internal/project/common"
	"github.com/siderolabs/kres/internal/project/golang"
)

// DetectIntegrationTests checks if the Go project has any integration tests.
func (builder *builder) DetectIntegrationTests() (bool, error) {
	var integrationTests IntegrationTests

	if err := builder.meta.Config.Load(&integrationTests); err != nil {
		return false, err
	}

	return len(integrationTests.Tests) > 0, nil
}

// BuildIntegrationTests builds Go integration tests.
func (builder *builder) BuildIntegrationTests() error {
	var integrationTests IntegrationTests

	if err := builder.meta.Config.Load(&integrationTests); err != nil {
		return err
	}

	for _, spec := range integrationTests.Tests {
		build := golang.NewBuild(builder.meta, spec.Name, spec.Path, "go test -c -covermode=atomic")

		build.Outputs = maps.Map(spec.Outputs, func(k string, m map[string]string) (string, golang.CompileConfig) {
			return k, golang.CompileConfig(m)
		})

		build.BuildFlags = append(build.BuildFlags, "-tags integration,sidero.debug")

		builder.targets = append(builder.targets, build)

		if spec.EnableDockerImage {
			imageName := spec.Name

			if spec.ImageName != "" {
				imageName = spec.ImageName
			}

			image := common.NewImage(
				builder.meta, imageName,
			)

			image.AddInput(build)

			builder.targets = append(builder.targets, image)
		}
	}

	return nil
}

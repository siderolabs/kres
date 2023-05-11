// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package auto

import (
	"fmt"

	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/project/custom"
)

// DetectCustom checks if project has any custom steps.
func (builder *builder) DetectCustom() (bool, error) {
	var customSteps CustomSteps

	if err := builder.meta.Config.Load(&customSteps); err != nil {
		return false, err
	}

	return len(customSteps.Steps) > 0, nil
}

// BuildCustom builds custom project steps.
func (builder *builder) BuildCustom() error {
	var customSteps CustomSteps

	if err := builder.meta.Config.Load(&customSteps); err != nil {
		return err
	}

	createdSteps := make([]dag.Node, 0, len(customSteps.Steps))

	for _, spec := range customSteps.Steps {
		step := custom.NewStep(builder.meta, spec.Name)

		if spec.Toplevel {
			builder.targets = append(builder.targets, step)
		}

		for _, inputName := range spec.Inputs {
			input := dag.FindByName(inputName, append(builder.targets, createdSteps...)...)

			if input == nil {
				return fmt.Errorf("failed to find input node %q for custom step %q", inputName, spec.Name)
			}

			step.AddInput(input)
		}

		createdSteps = append(createdSteps, step)
	}

	return nil
}

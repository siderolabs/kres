// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package auto

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/siderolabs/kres/internal/project/helm"
)

// DetectHelm checks the helm settings.
// It returns true if helm is enabled and the chart path is set.
func (builder *builder) DetectHelm() (bool, error) {
	var helm Helm

	if err := builder.meta.Config.Load(&helm); err != nil {
		return false, err
	}

	if !helm.Enabled {
		return false, nil
	}

	if helm.ChartDir == "" {
		return false, fmt.Errorf("chart directory is not set")
	}

	if _, err := os.Stat(filepath.Join(builder.rootPath, helm.ChartDir, "Chart.yaml")); err != nil {
		return false, fmt.Errorf("chart.yaml not found in %s: %w", helm.ChartDir, err)
	}

	builder.meta.HelmChartDir = helm.ChartDir

	if helm.E2EDir == "" {
		helm.E2EDir = filepath.Join(filepath.Dir(helm.ChartDir), "e2e")
	}

	builder.meta.HelmE2EDir = helm.E2EDir

	var flags []string

	for _, flag := range helm.Template.Set {
		flags = append(flags, "--set", flag)
	}

	for _, flag := range helm.Template.SetFile {
		flags = append(flags, "--set-file", flag)
	}

	for _, flag := range helm.Template.SetJSON {
		flags = append(flags, "--set-json", flag)
	}

	for _, flag := range helm.Template.SetLiteral {
		flags = append(flags, "--set-literal", flag)
	}

	for _, flag := range helm.Template.SetString {
		flags = append(flags, "--set-string", flag)
	}

	builder.meta.HelmTemplateFlags = flags

	return true, nil
}

func (builder *builder) BuildHelm() error {
	helm := helm.NewBuild(builder.meta)

	builder.targets = append(builder.targets, helm)
	helm.AddInput(builder.commonInputs...)

	return nil
}

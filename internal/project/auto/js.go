// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package auto

import (
	"os"
	"path/filepath"

	"github.com/talos-systems/kres/internal/project/js"
)

// DetectJS checks if project at rootPath/frontend is JS-based project.
func (builder *builder) DetectJS() (bool, error) {
	jsRoot := filepath.Join(builder.rootPath, "frontend")

	npmPackagePath := filepath.Join(jsRoot, "package.json")

	npmPackageConfig, err := os.Open(npmPackagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	defer npmPackageConfig.Close() //nolint: errcheck

	for _, srcDir := range []string{"frontend"} {
		exists, err := directoryExists(builder.rootPath, srcDir)
		if err != nil {
			return false, err
		}

		for _, path := range []string{"src", "tests", "public"} {
			d := filepath.Join(srcDir, path)

			if exists {
				builder.meta.Directories = append(builder.meta.Directories, d)
				builder.meta.JSDirectories = append(builder.meta.JSDirectories, d)
			}
		}

		results, err := listFilesWithSuffix(srcDir, ".js")
		if err != nil {
			return false, err
		}

		for _, item := range results {
			builder.meta.JSDirectories = append(builder.meta.JSDirectories, filepath.Join(srcDir, item))
		}

		builder.meta.SourceFiles = append(builder.meta.SourceFiles,
			filepath.Join(srcDir, "*.json"),
			filepath.Join(srcDir, "*.js"),
		)
	}

	return true, nil
}

// BuildJS builds project structure for JS project.
func (builder *builder) BuildJS() error {
	// toolchain as the root of the tree
	toolchain := js.NewToolchain(builder.meta, "frontend")
	toolchain.AddInput(builder.commonInputs...)

	// unit-tests
	unitTests := js.NewUnitTests(builder.meta, "unit-tests-frontend")
	unitTests.AddInput(toolchain)
	builder.targets = append(builder.targets, unitTests)

	// linters
	esLint := js.NewEsLint(builder.meta)
	esLint.AddInput(toolchain)
	builder.targets = append(builder.targets, esLint)

	build := js.NewBuild(builder.meta, "frontend")
	build.AddInput(toolchain)
	builder.targets = append(builder.targets, build)
	builder.commonInputs = append(builder.commonInputs, build)

	return nil
}

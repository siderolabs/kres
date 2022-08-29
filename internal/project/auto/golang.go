// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package auto

import (
	"io"
	"os"
	"path"
	"path/filepath"

	"golang.org/x/mod/modfile"

	"github.com/siderolabs/kres/internal/project/common"
	"github.com/siderolabs/kres/internal/project/golang"
	"github.com/siderolabs/kres/internal/project/service"
	"github.com/siderolabs/kres/internal/project/wrap"
)

// DetectGolang checks if project at rootPath is Go-based project.
//
//nolint:gocognit,gocyclo,cyclop
func (builder *builder) DetectGolang() (bool, error) {
	gomodPath := filepath.Join(builder.rootPath, "go.mod")

	gomod, err := os.Open(gomodPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	defer gomod.Close() //nolint:errcheck

	contents, err := io.ReadAll(gomod)
	if err != nil {
		return true, err
	}

	builder.meta.CanonicalPath = modfile.ModulePath(contents)

	for _, srcDir := range []string{
		"api",         // API definitions (generated protobufs, Kubebuilder's resources)
		"cmd",         // main packages
		"controllers", // Kubernetes controllers
		"internal",    // internal packages
		"pkg",         // generic, general use packages that can be used independently
		"src",         // deprecated
	} {
		exists, err := directoryExists(builder.rootPath, srcDir)
		if err != nil {
			return true, err
		}

		if exists {
			builder.meta.Directories = append(builder.meta.Directories, srcDir)
			builder.meta.GoDirectories = append(builder.meta.GoDirectories, srcDir)
		}
	}

	if len(builder.meta.GoDirectories) == 0 {
		// no standard directories found, assume any directory with `.go` files is a source directory
		topLevel, err := os.ReadDir(builder.rootPath)
		if err != nil {
			return true, err
		}

		for _, item := range topLevel {
			if !item.IsDir() {
				continue
			}

			result, err := listFilesWithSuffix(filepath.Join(builder.rootPath, item.Name()), ".go")
			if err != nil {
				return true, err
			}

			if len(result) > 0 {
				builder.meta.Directories = append(builder.meta.Directories, item.Name())
				builder.meta.GoDirectories = append(builder.meta.GoDirectories, item.Name())
			}
		}
	}

	{
		list, err := listFilesWithSuffix(builder.rootPath, ".go")
		if err != nil {
			return true, err
		}

		for _, item := range list {
			builder.meta.SourceFiles = append(builder.meta.SourceFiles, item)
			builder.meta.GoSourceFiles = append(builder.meta.GoSourceFiles, item)
		}
	}

	builder.meta.SourceFiles = append(builder.meta.SourceFiles, "go.mod", "go.sum")

	for _, candidate := range []string{"pkg/version", "internal/version"} {
		exists, err := directoryExists(builder.rootPath, candidate)
		if err != nil {
			return true, err
		}

		if exists {
			builder.meta.VersionPackage = path.Join(builder.meta.CanonicalPath, candidate)
		}
	}

	{
		cmdExists, err := directoryExists(builder.rootPath, "cmd")
		if err != nil {
			return true, err
		}

		if cmdExists {
			dirs, err := os.ReadDir(filepath.Join(builder.rootPath, "cmd"))
			if err != nil {
				return true, err
			}

			for _, dir := range dirs {
				if dir.IsDir() {
					builder.meta.Commands = append(builder.meta.Commands, dir.Name())
				}
			}
		}
	}

	return true, nil
}

// BuildGolang builds project structure for Go project.
func (builder *builder) BuildGolang() error {
	// toolchain as the root of the tree
	toolchain := golang.NewToolchain(builder.meta)
	toolchain.AddInput(builder.commonInputs...)

	// linters
	golangciLint := golang.NewGolangciLint(builder.meta)
	gofumpt := golang.NewGofumpt(builder.meta)
	goimports := golang.NewGoimports(builder.meta)

	// linters are input to the toolchain as they inject into toolchain build
	toolchain.AddInput(golangciLint, gofumpt, goimports)

	// add protobufs and go generate
	generate := golang.NewGenerate(builder.meta)

	// add deepcopy
	deepcopy := golang.NewDeepCopy(builder.meta)

	toolchain.AddInput(generate, deepcopy)

	builder.lintInputs = append(builder.lintInputs, toolchain, golangciLint, gofumpt, goimports)

	// unit-tests
	unitTests := golang.NewUnitTests(builder.meta)
	unitTests.AddInput(toolchain)

	coverage := service.NewCodeCov(builder.meta)
	coverage.InputPath = "coverage.txt"
	coverage.AddInput(unitTests)

	builder.targets = append(builder.targets, unitTests, coverage)

	// process commands
	for _, cmd := range builder.meta.Commands {
		cfg := CommandConfig{NamedConfig: NamedConfig{name: cmd}}
		if err := builder.meta.Config.Load(&cfg); err != nil {
			return err
		}

		build := golang.NewBuild(builder.meta, cmd, filepath.Join("cmd", cmd))
		build.AddInput(toolchain)
		builder.targets = append(builder.targets, build)

		if !cfg.DisableImage {
			image := common.NewImage(builder.meta, cmd)
			image.AddInput(build, builder.lintTarget, wrap.Drone(unitTests))

			builder.targets = append(builder.targets, image)
		}
	}

	return nil
}

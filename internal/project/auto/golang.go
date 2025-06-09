// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package auto

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"golang.org/x/mod/modfile"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/project/common"
	"github.com/siderolabs/kres/internal/project/golang"
	"github.com/siderolabs/kres/internal/project/meta"
	"github.com/siderolabs/kres/internal/project/service"
	"github.com/siderolabs/kres/internal/project/wrap"
)

// DetectGolang checks if project at rootPath is Go-based project.
func (builder *builder) DetectGolang() (bool, error) {
	// skip detecting additional go build steps if Pkgfile is detected
	if builder.meta.ContainerImageFrontend != config.ContainerImageFrontendDockerfile {
		return false, nil
	}

	var lookupDirs []string

	err := filepath.Walk(builder.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Base(path) == "go.mod" {
			lookupDirs = append(lookupDirs, filepath.Dir(path))
		}

		return nil
	})
	if err != nil {
		return false, err
	}

	for _, dir := range lookupDirs {
		_, err := os.Stat(filepath.Join(dir, ".kresignore"))
		if err == nil {
			continue
		}

		if !os.IsNotExist(err) {
			return false, err
		}

		if err := builder.processDirectory(dir); err != nil {
			return false, err
		}

		builder.meta.GoRootDirectories = append(builder.meta.GoRootDirectories, dir)
	}

	if len(builder.meta.GoSourceFiles) == 0 && len(builder.meta.GoDirectories) == 0 {
		return false, errors.New("no Go source files found")
	}

	return true, nil
}

//nolint:gocognit,gocyclo,cyclop
func (builder *builder) processDirectory(path string) error {
	var (
		canonicalPath string
		dir           = filepath.Join(builder.rootPath, path)
	)

	{
		gomodPath := filepath.Join(dir, "go.mod")

		gomod, err := os.Open(gomodPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}

			return err
		}

		defer gomod.Close() //nolint:errcheck

		contents, err := io.ReadAll(gomod)
		if err != nil {
			return err
		}

		canonicalPath = modfile.ModulePath(contents)
	}

	builder.meta.CanonicalPaths = append(builder.meta.CanonicalPaths, canonicalPath)

	for _, srcDir := range []string{
		"api",         // API definitions (generated protobufs, Kubebuilder's resources)
		"cmd",         // main packages
		"controllers", // Kubernetes controllers
		"internal",    // internal packages
		"pkg",         // generic, general use packages that can be used independently
		"src",         // deprecated
	} {
		exists, err := directoryExists(dir, srcDir)
		if err != nil {
			return err
		}

		if exists {
			builder.meta.Directories = append(builder.meta.Directories, filepath.Join(dir, srcDir))
			builder.meta.GoDirectories = append(builder.meta.GoDirectories, filepath.Join(dir, srcDir))
		}
	}

	{
		// assume any directory with `.go` files is a source directory
		topLevel, err := os.ReadDir(dir)
		if err != nil {
			return err
		}

		for _, item := range topLevel {
			if !item.IsDir() {
				continue
			}

			if slices.Index(builder.meta.Directories, filepath.Join(dir, item.Name())) != -1 {
				continue
			}

			result, err := listFilesWithSuffix(filepath.Join(dir, item.Name()), ".go")
			if err != nil {
				return err
			}

			if len(result) > 0 {
				builder.meta.Directories = append(builder.meta.Directories, filepath.Join(dir, item.Name()))
				builder.meta.GoDirectories = append(builder.meta.GoDirectories, filepath.Join(dir, item.Name()))
			}
		}
	}

	{
		list, err := listFilesWithSuffix(dir, ".go")
		if err != nil {
			return err
		}

		for _, item := range list {
			builder.meta.SourceFiles = append(builder.meta.SourceFiles, filepath.Join(dir, item))
			builder.meta.GoSourceFiles = append(builder.meta.GoSourceFiles, filepath.Join(dir, item))
		}
	}

	builder.meta.SourceFiles = append(builder.meta.SourceFiles,
		filepath.Join(dir, "go.mod"),
		filepath.Join(dir, "go.sum"),
	)

	rootPath := filepath.Join(builder.rootPath, dir)

	if builder.meta.VersionPackagePath == "" {
		for _, candidate := range []string{"pkg/version", "internal/version"} {
			exists, err := directoryExists(dir, candidate)
			if err != nil {
				return err
			} else if !exists {
				continue
			}

			list, err := listFilesWithSuffix(filepath.Join(dir, candidate), ".go")
			if err != nil {
				return err
			}

			if len(list) == 0 || !slices.Contains(list, "version.go") {
				continue
			}

			builder.meta.VersionPackagePath = filepath.Join(canonicalPath, filepath.Join(dir, candidate))
		}
	}

	cmdExists, err := directoryExists(rootPath, "cmd")
	if err != nil {
		return err
	}

	if cmdExists {
		path := filepath.Join(dir, "cmd")

		dirs, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		for _, dir := range dirs {
			if dir.IsDir() {
				builder.meta.Commands = append(builder.meta.Commands, meta.Command{
					Path: filepath.Join(path, dir.Name()),
					Name: dir.Name(),
				})
			}
		}
	}

	return nil
}

// BuildGolang builds project structure for Go project.
func (builder *builder) BuildGolang() error {
	// toolchain as the root of the tree
	toolchain := golang.NewToolchain(builder.meta)
	toolchain.AddInput(builder.commonInputs...)

	// add protobufs and go generate
	generate := golang.NewGenerate(builder.meta)

	// add deepcopy
	deepcopy := golang.NewDeepCopy(builder.meta)

	// add common linter tools
	linters := golang.NewLinters(builder.meta)

	toolchain.AddInput(generate, deepcopy, linters)

	builder.lintInputs = append(builder.lintInputs, toolchain, linters)

	coverage := service.NewCodeCov(builder.meta)
	allUnitTests := make([]dag.Node, 0, len(builder.meta.CanonicalPaths))

	// linters
	for _, projectPath := range builder.meta.GoRootDirectories {
		golangciLint := golang.NewGolangciLint(builder.meta, projectPath)
		gofumpt := golang.NewGofumpt(builder.meta, projectPath)
		govulncheck := golang.NewGoVulnCheck(builder.meta, projectPath)

		// linters are input to the toolchain as they inject into toolchain build
		toolchain.AddInput(golangciLint, gofumpt, govulncheck)

		builder.lintInputs = append(builder.lintInputs, toolchain, golangciLint, gofumpt, govulncheck)

		// unit-tests
		unitTests := golang.NewUnitTests(builder.meta, projectPath)
		unitTests.AddInput(toolchain)

		coverage.AddInput(unitTests)

		builder.targets = append(builder.targets, unitTests)
		allUnitTests = append(allUnitTests, unitTests)

		coverage.AddDiscoveredInputs(fmt.Sprintf("coverage-%s.txt", unitTests.Name()))
	}

	builder.targets = append(builder.targets, coverage)

	// process commands
	for _, cmd := range builder.meta.Commands {
		cfg := CommandConfig{NamedConfig: NamedConfig{name: cmd.Name}}
		if err := builder.meta.Config.Load(&cfg); err != nil {
			return err
		}

		build := golang.NewBuild(builder.meta, cmd.Name, cmd.Path, "go build")
		build.AddInput(toolchain)
		builder.targets = append(builder.targets, build)

		if !cfg.DisableImage {
			image := common.NewImage(builder.meta, cmd.Name)

			for _, unitTests := range allUnitTests {
				image.AddInput(build, builder.lintTarget, wrap.Drone(unitTests))
			}

			builder.targets = append(builder.targets, image)
		}
	}

	return nil
}

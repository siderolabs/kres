// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package meta provides project options from source code.
package meta

import (
	"slices"

	"github.com/siderolabs/kres/internal/config"
)

// Options for the project.
type Options struct { //nolint:govet
	// Config provider.
	Config *config.Provider

	// GitHub settings.
	GitHubOrganization string
	GitHubRepository   string

	// Git settings.
	MainBranch string

	// CanonicalPaths, import path for Go projects.
	CanonicalPaths []string

	// VersionPackagePath is a canonical path to version package directory.
	VersionPackagePath string

	// Directories which contain source code.
	Directories []string

	// JSDirectories which contain JS source code.
	JSDirectories []string

	// GoDirectories are directories containing Go source code.
	GoDirectories []string

	// ProtobufDirectories are directories containing .proto files.
	ProtobufDirectories []string

	// MarkdownDirectories are directories container Markdown files.
	MarkdownDirectories []string

	// Source files on top level.
	SourceFiles []string

	// Go source files on top level.
	GoSourceFiles []string

	// Markdown source files on top level.
	MarkdownSourceFiles []string

	// Commands are top-level binaries to be built.
	Commands []Command

	// GoRootDirectories contans the list of all go.mod root directories.
	GoRootDirectories []string

	// BuildArgs passed down to Dockerfiles.
	BuildArgs BuildArgs

	// Path to /bin.
	BinPath string

	// GoContainerVersion is the default go official container version.
	GoContainerVersion string

	// Go's GOPATH.
	GoPath string

	// Path to ~/.cache.
	CachePath string

	// JSCachePath path to ~/.npm.
	JSCachePath string

	// ArtifactsPath binary output path.
	ArtifactsPath string

	// CIProvider specifies the CI provider. Currently drone/ghaction is supported.
	CIProvider string

	// CompileGithubWorkflowsOnly indicates that only GitHub workflows should be compiled.
	CompileGithubWorkflowsOnly bool

	// EnforcedContexts is the list of required status checks for GitHub branch protection.
	ExtraEnforcedContexts []string

	// ContainerImageFrontend is the default frontend container image.
	ContainerImageFrontend string

	// HelmChartDir is the path to helm chart directory.
	HelmChartDir string

	// SkipStaleWorkflow indicates that stale workflow should not be generated.
	SkipStaleWorkflow bool

	// CIFailureSlackNotifyChannel is the Slack channel to notify on CI failures.
	CIFailureSlackNotifyChannel string

	// SOPSEnabled indicates whether SOPS is enabled for the project.
	SOPSEnabled bool
}

// Command defines Golang executable build configuration.
type Command struct {
	// Path defines command source path.
	Path string

	// Name defines command name.
	Name string
}

// BuildArgs defines input argument list.
type BuildArgs []string

// Add adds the args to list if it doesn't exist already.
func (args *BuildArgs) Add(arg ...string) {
	for _, value := range arg {
		if slices.Contains(*args, value) {
			continue
		}

		*args = append(*args, value)
	}
}

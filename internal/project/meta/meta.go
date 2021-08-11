// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package meta provides project options from source code.
package meta

import "github.com/talos-systems/kres/internal/config"

// Options for the project.
type Options struct { //nolint:govet
	// Config provider.
	Config *config.Provider

	// GitHub settings.
	GitHubOrganization string
	GitHubRepository   string

	// CanonicalPath, import path for Go projects.
	CanonicalPath string

	// VersionPackage is a canonical path to version package (if any).
	VersionPackage string

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

	// JS source files on top level.
	JSSourceFiles []string

	// Markdown source files on top level.
	MarkdownSourceFiles []string

	// Commands are top-level binaries to be built.
	Commands []string

	// BuildArgs passed down to Dockerfiles.
	BuildArgs []string

	// Path to /bin.
	BinPath string

	// Go's GOPATH.
	GoPath string

	// Path to ~/.cache.
	CachePath string

	// NpmCachePath path to node_modules.
	NpmCachePath string

	// ArtifactsPath binary output path.
	ArtifactsPath string
}

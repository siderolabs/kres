// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package config provides config loading and mapping.
package config

const (
	// CIProviderDrone is the drone ci provider.
	CIProviderDrone = "drone"
	// CIProviderGitHubActions is the github actions ci provider.
	CIProviderGitHubActions = "ghaction"
	// ContainerImageFrontendDockerfile is the dockerfile frontend.
	ContainerImageFrontendDockerfile = "Dockerfile"
	// ContainerImageFrontendPkgfile is the pkgfile frontend.
	ContainerImageFrontendPkgfile = "Pkgfile"

	// BldrImageVersion is the version of bldr image.
	// renovate: datasource=github-releases depName=siderolabs/bldr
	BldrImageVersion = "v0.4.1"

	// CheckOutActionVersion is the version of checkout github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=actions/checkout
	CheckOutActionVersion = "v4"
	// CodeCovActionVersion is the version of codecov github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=codecov/codecov-action
	CodeCovActionVersion = "v5"
	// DeepCopyVersion is the version of deepcopy.
	// renovate: datasource=go depName=github.com/siderolabs/deep-copy
	DeepCopyVersion = "v0.5.6"
	// DindContainerImageVersion is the version of the dind container image.
	// renovate: datasource=docker versioning=docker depName=docker
	DindContainerImageVersion = "28.1-dind"
	// DockerfileFrontendImageVersion is the version of the dockerfile frontend image.
	// renovate: datasource=docker versioning=docker depName=docker/dockerfile-upstream
	DockerfileFrontendImageVersion = "1.15.1-labs"
	// DownloadArtifactActionVersion is the version of download artifact github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=actions/download-artifact
	DownloadArtifactActionVersion = "v4"
	// GitHubScriptActionVersion is the version of github script action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=actions/github-script
	GitHubScriptActionVersion = "v7"
	// GoFmtVersion is the version of gofmt.
	// renovate: datasource=go depName=github.com/mvdan/gofumpt
	GoFmtVersion = "v0.8.0"
	// GoImportsVersion is the version of goimports.
	// renovate: datasource=go depName=golang.org/x/tools
	GoImportsVersion = "v0.32.0"
	// GoMockVersion is the version of gomock.
	// renovate: datasource=go depName=github.com/uber-go/mock
	GoMockVersion = "v0.5.2"
	// GolangCIlintVersion is the version of golangci-lint.
	// renovate: datasource=go depName=github.com/golangci/golangci-lint
	GolangCIlintVersion = "v2.1.5"
	// GolangContainerImageVersion is the default golang container image.
	// renovate: datasource=docker versioning=docker depName=golang
	GolangContainerImageVersion = "1.24-alpine"
	// GoVersion is the version of Go.
	// renovate: datasource=github-tags extractVersion=^go(?<version>.*)$ depName=golang/go
	GoVersion = "1.24.2"
	// GrpcGatewayVersion is the version of grpc-gateway.
	// renovate: datasource=go depName=github.com/grpc-ecosystem/grpc-gateway
	GrpcGatewayVersion = "v2.26.3"
	// GrpcGoVersion is the version of grpc.
	// renovate: datasource=go depName=google.golang.org/grpc/cmd/protoc-gen-go-grpc
	GrpcGoVersion = "v1.5.1"
	// LoginActionVersion is the version of login github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=docker/login-action
	LoginActionVersion = "v3"
	// MarkdownLintCLIVersion is the version of markdownlint.
	// renovate: datasource=npm depName=markdownlint-cli
	MarkdownLintCLIVersion = "0.44.0"
	// BunContainerImageVersion is the default bun container image.
	// renovate: datasource=docker versioning=docker depName=oven/bun
	BunContainerImageVersion = "1.2.11-alpine"
	// PkgsVersion is the version of pkgs.
	// renovate: datasource=github-tags depName=siderolabs/pkgs
	PkgsVersion = "v1.10.0"
	// ProtobufGoVersion is the version of protobuf.
	// renovate: datasource=go depName=google.golang.org/protobuf/cmd/protoc-gen-go
	ProtobufGoVersion = "v1.36.6"
	// ProtobufTSGatewayVersion is the version of protobuf-ts.
	// renovate: datasource=go depName=github.com/siderolabs/protoc-gen-grpc-gateway-ts
	ProtobufTSGatewayVersion = "v1.2.1"
	// ReleaseActionVersion is the version of release github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=crazy-max/ghaction-github-release
	ReleaseActionVersion = "v2"
	// SentencesPerLineVersion is the version of sentences-per-line.
	// renovate: datasource=npm depName=sentences-per-line
	SentencesPerLineVersion = "0.3.0"
	// SetupBuildxActionVersion is the version of setup-buildx github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=docker/setup-buildx-action
	SetupBuildxActionVersion = "v3"
	// SetupTerraformActionVersion is the version of setup terraform github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=hashicorp/setup-terraform
	SetupTerraformActionVersion = "v3"
	// SlackNotifyActionVersion is the version of slack notify github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=slackapi/slack-github-action
	SlackNotifyActionVersion = "v2"
	// SystemInfoActionVersion is the version of system info github action.
	// renovate: datasource=github-releases depName=kenchan0130/actions-system-info
	SystemInfoActionVersion = "v1.3.0"
	// UploadArtifactActionVersion is the version of upload artifact github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=actions/upload-artifact
	UploadArtifactActionVersion = "v4"
	// VTProtobufVersion is the version of vtprotobuf.
	// renovate: datasource=go depName=github.com/planetscale/vtprotobuf
	VTProtobufVersion = "v0.6.0"
)

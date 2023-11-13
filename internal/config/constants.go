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
	BldrImageVersion = "v0.2.3"

	// BuildKitContainerVersion is the version of buildkit container image.
	// renovate: datasource=docker versioning=docker depName=moby/buildkit
	BuildKitContainerVersion = "v0.12.3"
	// CheckOutActionVersion is the version of checkout github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=actions/checkout
	CheckOutActionVersion = "v4"
	// DeepCopyVersion is the version of deepcopy.
	// renovate: datasource=go depName=github.com/siderolabs/deep-copy
	DeepCopyVersion = "v0.5.5"
	// DindContainerImageVersion is the version of the dind container image.
	// renovate: datasource=docker versioning=docker depName=docker
	DindContainerImageVersion = "24.0-dind"
	// DockerfileFrontendImageVersion is the version of the dockerfile frontend image.
	// renovate: datasource=docker versioning=docker depName=docker/dockerfile-upstream
	DockerfileFrontendImageVersion = "1.6.0-labs"
	// DownloadArtifactActionVersion is the version of download artifact github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=actions/download-artifact
	DownloadArtifactActionVersion = "v3"
	// GitHubScriptActionVersion is the version of github script action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=actions/github-script
	GitHubScriptActionVersion = "v6"
	// GoFmtVersion is the version of gofmt.
	// renovate: datasource=go depName=github.com/mvdan/gofumpt
	GoFmtVersion = "v0.5.0"
	// GoImportsVersion is the version of goimports.
	// renovate: datasource=go depName=golang.org/x/tools
	GoImportsVersion = "v0.15.0"
	// GolangCIlintVersion is the version of golangci-lint.
	// renovate: datasource=go depName=github.com/golangci/golangci-lint
	GolangCIlintVersion = "v1.55.2"
	// GolangContainerImageVersion is the default golang container image.
	// renovate: datasource=docker versioning=docker depName=golang
	GolangContainerImageVersion = "1.21-alpine"
	// GoVersion is the version of Go.
	// renovate: datasource=github-tags extractVersion=^go(?<version>.*)$ depName=golang/go
	GoVersion = "1.21.4"
	// GrpcGatewayVersion is the version of grpc-gateway.
	// renovate: datasource=go depName=github.com/grpc-ecosystem/grpc-gateway
	GrpcGatewayVersion = "v2.18.1"
	// GrpcGoVersion is the version of grpc.
	// renovate: datasource=go depName=google.golang.org/grpc/cmd/protoc-gen-go-grpc
	GrpcGoVersion = "v1.3.0"
	// LoginActionVersion is the version of login github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=docker/login-action
	LoginActionVersion = "v3"
	// MardownLintCLIVersion is the version of markdownlint.
	// renovate: datasource=npm depName=markdownlint-cli
	MardownLintCLIVersion = "0.37.0"
	// NodeContainerImageVersion is the default node container image.
	// renovate: datasource=docker versioning=docker depName=node
	NodeContainerImageVersion = "21.1.0-alpine3.18"
	// PkgsVersion is the version of pkgs.
	// renovate: datasource=github-tags depName=siderolabs/pkgs
	PkgsVersion = "v1.6.0-alpha.0-10-gd3d7d29"
	// ProtobufGoVersion is the version of protobuf.
	// renovate: datasource=go depName=google.golang.org/protobuf/cmd/protoc-gen-go
	ProtobufGoVersion = "v1.31.0"
	// ProtobufTSGatewayVersion is the version of protobuf-ts.
	// renovate: datasource=go depName=github.com/siderolabs/protoc-gen-grpc-gateway-ts
	ProtobufTSGatewayVersion = "v1.2.1"
	// ReleaseActionVersion is the version of release github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=crazy-max/ghaction-github-release
	ReleaseActionVersion = "v2"
	// SentencesPerLineVersion is the version of sentences-per-line.
	// renovate: datasource=npm depName=sentences-per-line
	SentencesPerLineVersion = "0.2.1"
	// SetupBuildxActionVersion is the version of setup-buildx github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=docker/setup-buildx-action
	SetupBuildxActionVersion = "v3"
	// SlackNotifyActionVersion is the version of slack notify github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=slackapi/slack-github-action
	SlackNotifyActionVersion = "v1"
	// UploadArtifactActionVersion is the version of upload artifact github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=actions/upload-artifact
	UploadArtifactActionVersion = "v3"
	// VTProtobufVersion is the version of vtprotobuf.
	// renovate: datasource=go depName=github.com/planetscale/vtprotobuf
	VTProtobufVersion = "v0.5.0"
)

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
	// CheckOutActionVersion is the version of checkout github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=actions/checkout
	CheckOutActionVersion = "v3"
	// DeepCopyVersion is the version of deepcopy.
	// renovate: datasource=go depName=github.com/siderolabs/deep-copy
	DeepCopyVersion = "v0.5.5"
	// DindContainerImageVersion is the version of the dind container image.
	// renovate: datasource=docker versioning=docker depName=docker
	DindContainerImageVersion = "24.0-dind"
	// DockerfileFrontendImageVersion is the version of the dockerfile frontend image.
	// renovate: datasource=docker versioning=docker depName=docker/dockerfile-upstream
	DockerfileFrontendImageVersion = "1.6.0-labs"
	// GoFmtVersion is the version of gofmt.
	// renovate: datasource=go depName=github.com/mvdan/gofumpt
	GoFmtVersion = "v0.5.0"
	// GoImportsVersion is the version of goimports.
	// renovate: datasource=go depName=golang.org/x/tools
	GoImportsVersion = "v0.13.0"
	// GolangCIlintVersion is the version of golangci-lint.
	// renovate: datasource=go depName=github.com/golangci/golangci-lint
	GolangCIlintVersion = "v1.54.2"
	// GolangContainerImageVersion is the default golang container image.
	// renovate: datasource=docker versioning=docker depName=golang
	GolangContainerImageVersion = "1.21-alpine"
	// GoVersion is the version of Go.
	// renovate: datasource=github-tags extractVersion=^go(?<version>.*)$ depName=golang/go
	GoVersion = "1.21"
	// GrpcGatewayVersion is the version of grpc-gateway.
	// renovate: datasource=go depName=github.com/grpc-ecosystem/grpc-gateway
	GrpcGatewayVersion = "v2.18.0"
	// GrpcGoVersion is the version of grpc.
	// renovate: datasource=go depName=google.golang.org/grpc/cmd/protoc-gen-go-grpc
	GrpcGoVersion = "v1.3.0"
	// LoginActionVersion is the version of login github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=docker/login-action
	LoginActionVersion = "v2"
	// MardownLintCLIVersion is the version of markdownlint.
	// renovate: datasource=npm depName=markdownlint-cli
	MardownLintCLIVersion = "0.35.0"
	// NodeContainerImageVersion is the default node container image.
	// renovate: datasource=docker versioning=docker depName=node
	NodeContainerImageVersion = "20.5.1-alpine3.18"
	// PkgsVersion is the version of pkgs.
	// renovate: datasource=github-tags depName=siderolabs/pkgs
	PkgsVersion = "v1.6.0-alpha.0-10-gd3d7d29"
	// ProtobufGoVersion is the version of protobuf.
	// renovate: datasource=go depName=google.golang.org/protobuf/cmd/protoc-gen-go
	ProtobufGoVersion = "v1.31.0"
	// ProtobufTSGatewayVersion is the version of protobuf-ts.
	// renovate: datasource=go depName=github.com/siderolabs/protoc-gen-grpc-gateway-ts
	ProtobufTSGatewayVersion = "v1.2.0"
	// ReleaseActionVersion is the version of release github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=crazy-max/ghaction-github-release
	ReleaseActionVersion = "v1"
	// SentencesPerLineVersion is the version of sentences-per-line.
	// renovate: datasource=npm depName=sentences-per-line
	SentencesPerLineVersion = "0.2.1"
	// SetupBuildxActionVersion is the version of setup-buildx github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=docker/setup-buildx-action
	SetupBuildxActionVersion = "v2"
	// SlackNotifyActionVersion is the version of slack notify github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=slackapi/slack-github-action
	SlackNotifyActionVersion = "v1"
	// VTProtobufVersion is the version of vtprotobuf.
	// renovate: datasource=go depName=github.com/planetscale/vtprotobuf
	VTProtobufVersion = "v0.5.0"
)

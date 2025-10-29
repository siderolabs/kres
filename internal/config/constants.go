// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

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
	BldrImageVersion = "v0.5.4"

	// CheckOutActionVersion is the version of checkout github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=actions/checkout
	CheckOutActionVersion = "v5"
	// CodeCovActionVersion is the version of codecov github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=codecov/codecov-action
	CodeCovActionVersion = "v5"
	// CosignInstallActionVerson is the version of cosign install github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=sigstore/cosign-installer
	CosignInstallActionVerson = "v3"
	// DeepCopyVersion is the version of deepcopy.
	// renovate: datasource=go depName=github.com/siderolabs/deep-copy
	DeepCopyVersion = "v0.5.8"
	// DindContainerImageVersion is the version of the dind container image.
	// renovate: datasource=docker versioning=docker depName=docker
	DindContainerImageVersion = "28.5-dind"
	// DockerfileFrontendImageVersion is the version of the dockerfile frontend image.
	// renovate: datasource=docker versioning=docker depName=docker/dockerfile-upstream
	DockerfileFrontendImageVersion = "1.19.0-labs"
	// DownloadArtifactActionVersion is the version of download artifact github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=actions/download-artifact
	DownloadArtifactActionVersion = "v4"
	// GitHubScriptActionVersion is the version of github script action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=actions/github-script
	GitHubScriptActionVersion = "v7"
	// GoFmtVersion is the version of gofmt.
	// renovate: datasource=go depName=github.com/mvdan/gofumpt
	GoFmtVersion = "v0.9.1"
	// GoImportsVersion is the version of goimports.
	// renovate: datasource=go depName=golang.org/x/tools
	GoImportsVersion = "v0.38.0"
	// GoMockVersion is the version of gomock.
	// renovate: datasource=go depName=github.com/uber-go/mock
	GoMockVersion = "v0.6.0"
	// GolangCIlintVersion is the version of golangci-lint.
	// renovate: datasource=go depName=github.com/golangci/golangci-lint
	GolangCIlintVersion = "v2.5.0"
	// GolangContainerImageVersion is the default golang container image.
	// renovate: datasource=docker versioning=docker depName=golang
	GolangContainerImageVersion = "1.25-alpine"
	// GoVersion is the version of Go.
	// renovate: datasource=github-tags extractVersion=^go(?<version>.*)$ depName=golang/go
	GoVersion = "1.25.3"
	// GrpcGatewayVersion is the version of grpc-gateway.
	// renovate: datasource=go depName=github.com/grpc-ecosystem/grpc-gateway
	GrpcGatewayVersion = "v2.27.3"
	// GrpcGoVersion is the version of grpc.
	// renovate: datasource=go depName=google.golang.org/grpc/cmd/protoc-gen-go-grpc
	GrpcGoVersion = "v1.5.1"
	// HelmSetupActionVersion is the version of helm setup github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=azure/setup-helm
	HelmSetupActionVersion = "v4"
	// LoginActionVersion is the version of login github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=docker/login-action
	LoginActionVersion = "v3"
	// MarkdownLintCLIVersion is the version of markdownlint.
	// renovate: datasource=npm depName=markdownlint-cli
	MarkdownLintCLIVersion = "0.45.0"
	// BunContainerImageVersion is the default bun container image.
	// renovate: datasource=docker versioning=docker depName=oven/bun
	BunContainerImageVersion = "1.3.0-alpine"
	// NodeContainerImageVersion is the default node container image.
	//
	// NOTE: Check renovate.json for the rules on this before bumping, e.g., pinned versions.
	// As a rule of thumb, we bump only to the versions promoted to be LTS (even [not odd] major versions get promoted after a while, always check).
	//
	// renovate: datasource=docker versioning=docker depName=node
	NodeContainerImageVersion = "24.11.0-alpine"
	// PkgsVersion is the version of pkgs.
	// renovate: datasource=github-tags depName=siderolabs/pkgs
	PkgsVersion = "v1.11.0"
	// ProtobufGoVersion is the version of protobuf.
	// renovate: datasource=go depName=google.golang.org/protobuf/cmd/protoc-gen-go
	ProtobufGoVersion = "v1.36.10"
	// ProtobufTSGatewayVersion is the version of protobuf-ts.
	// renovate: datasource=go depName=github.com/siderolabs/protoc-gen-grpc-gateway-ts
	ProtobufTSGatewayVersion = "v1.2.1"
	// ReleaseActionVersion is the version of release github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=softprops/action-gh-release
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
	SystemInfoActionVersion = "v1.4.0"
	// UploadArtifactActionVersion is the version of upload artifact github action.
	// renovate: datasource=github-releases extractVersion=^(?<version>v\d+)\.\d+\.\d+$ depName=actions/upload-artifact
	UploadArtifactActionVersion = "v4"
	// VTProtobufVersion is the version of vtprotobuf.
	// renovate: datasource=go depName=github.com/planetscale/vtprotobuf
	VTProtobufVersion = "v0.6.0"
	// StaleActionVersion is the version of stale github action.
	// renovate: datasource=github-releases depName=actions/stale
	StaleActionVersion = "v10.1.0"
	// LockThreadsActionVersion is the version of lock threads github action.
	// renovate: datasource=github-releases depName=dessant/lock-threads
	LockThreadsActionVersion = "v5.0.1"
)

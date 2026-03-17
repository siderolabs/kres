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
	BldrImageVersion = "v0.5.6"

	// CheckOutActionVersion is the version of checkout github action.
	// renovate: datasource=github-tags depName=actions/checkout
	CheckOutActionVersion = "v6.0.2"
	CheckOutActionRef     = "de0fac2e4500dabe0009e67214ff5f5447ce83dd"
	// CodeCovActionVersion is the version of codecov github action.
	// renovate: datasource=github-tags depName=codecov/codecov-action
	CodeCovActionVersion = "v5.5.2"
	CodeCovActionRef     = "671740ac38dd9b0130fbe1cec585b89eea48d3de"
	// CosignInstallActionVersion is the version of cosign install github action.
	// renovate: datasource=github-tags depName=sigstore/cosign-installer
	CosignInstallActionVersion = "v4.1.0"
	CosignInstallActionRef     = "ba7bc0a3fef59531c69a25acd34668d6d3fe6f22"
	// DeepCopyVersion is the version of deepcopy.
	// renovate: datasource=go depName=github.com/siderolabs/deep-copy
	DeepCopyVersion = "v0.5.8"
	// DindContainerImageVersion is the version of the dind container image.
	// renovate: datasource=docker versioning=docker depName=docker
	DindContainerImageVersion = "29.3-dind"
	// DockerfileFrontendImageVersion is the version of the dockerfile frontend image.
	// renovate: datasource=docker versioning=docker depName=docker/dockerfile-upstream
	DockerfileFrontendImageVersion = "1.22.0-labs"
	// DownloadArtifactActionVersion is the version of download artifact github action.
	// renovate: datasource=github-tags depName=actions/download-artifact
	DownloadArtifactActionVersion = "v8.0.0"
	DownloadArtifactActionRef     = "70fc10c6e5e1ce46ad2ea6f2b72d43f7d47b13c3"
	// GitHubScriptActionVersion is the version of github script action.
	// renovate: datasource=github-tags depName=actions/github-script
	GitHubScriptActionVersion = "v8.0.0"
	GitHubScriptActionRef     = "ed597411d8f924073f98dfc5c65a23a2325f34cd"
	// GoFmtVersion is the version of gofmt.
	// renovate: datasource=go depName=github.com/mvdan/gofumpt
	GoFmtVersion = "v0.9.2"
	// GoImportsVersion is the version of goimports.
	// renovate: datasource=go depName=golang.org/x/tools
	GoImportsVersion = "v0.42.0"
	// GoMockVersion is the version of gomock.
	// renovate: datasource=go depName=github.com/uber-go/mock
	GoMockVersion = "v0.6.0"
	// GolangCIlintVersion is the version of golangci-lint.
	// renovate: datasource=go depName=github.com/golangci/golangci-lint
	GolangCIlintVersion = "v2.11.3"
	// GolangContainerImageVersion is the default golang container image.
	// renovate: datasource=docker versioning=docker depName=golang
	GolangContainerImageVersion = "1.26-alpine"
	// GoVersion is the version of Go.
	// renovate: datasource=github-tags extractVersion=^go(?<version>.*)$ depName=golang/go
	GoVersion = "1.26.1"
	// GrpcGatewayVersion is the version of grpc-gateway.
	// renovate: datasource=go depName=github.com/grpc-ecosystem/grpc-gateway
	GrpcGatewayVersion = "v2.28.0"
	// GrpcGoVersion is the version of grpc.
	// renovate: datasource=go depName=google.golang.org/grpc/cmd/protoc-gen-go-grpc
	GrpcGoVersion = "v1.6.1"
	// HelmSetupActionVersion is the version of helm setup github action.
	// renovate: datasource=github-tags depName=Azure/setup-helm
	HelmSetupActionVersion = "v4.3.1"
	HelmSetupActionRef     = "1a275c3b69536ee54be43f2070a358922e12c8d4"
	// HelmUnitTestVersion is the version of helm unit test plugin.
	// renovate: datasource=github-tags depName=helm-unittest/helm-unittest
	HelmUnitTestVersion = "v1.0.3"
	// HelmValuesSchemaJSONVersion is the version of helm values-schema-json plugin.
	// renovate: datasource=github-tags depName=losisin/helm-values-schema-json
	HelmValuesSchemaJSONVersion = "v2.3.1"
	// HelmDocsVersion is the version of helm-docs tool.
	// renovate: datasource=github-tags depName=norwoodj/helm-docs
	HelmDocsVersion = "v1.14.2"
	// LoginActionVersion is the version of login github action.
	// renovate: datasource=github-tags depName=docker/login-action
	LoginActionVersion = "v4.0.0"
	LoginActionRef     = "b45d80f862d83dbcd57f89517bcf500b2ab88fb2"
	// MarkdownLintCLIVersion is the version of markdownlint.
	// renovate: datasource=npm depName=markdownlint-cli
	MarkdownLintCLIVersion = "0.48.0"
	// BunContainerImageVersion is the default bun container image.
	// renovate: datasource=docker versioning=docker depName=oven/bun
	BunContainerImageVersion = "1.3.10-alpine"
	// NodeContainerImageVersion is the default node container image.
	//
	// NOTE: Check renovate.json for the rules on this before bumping, e.g., pinned versions.
	// As a rule of thumb, we bump only to the versions promoted to be LTS (even [not odd] major versions get promoted after a while, always check).
	//
	// renovate: datasource=docker versioning=docker depName=node
	NodeContainerImageVersion = "24.14.0-alpine"
	// PkgsVersion is the version of pkgs.
	// renovate: datasource=github-tags depName=siderolabs/pkgs
	PkgsVersion = "v1.12.0"
	// ProtobufGoVersion is the version of protobuf.
	// renovate: datasource=go depName=google.golang.org/protobuf/cmd/protoc-gen-go
	ProtobufGoVersion = "v1.36.11"
	// ProtobufTSGatewayVersion is the version of protoc-gen-grpc-gateway-ts.
	// renovate: datasource=go depName=github.com/siderolabs/protoc-gen-grpc-gateway-ts
	ProtobufTSGatewayVersion = "v1.4.1"
	// ReleaseActionVersion is the version of release github action.
	// renovate: datasource=github-tags depName=softprops/action-gh-release
	ReleaseActionVersion = "v2.5.0"
	ReleaseActionRef     = "a06a81a03ee405af7f2048a818ed3f03bbf83c7b"
	// SentencesPerLineVersion is the version of sentences-per-line.
	// renovate: datasource=npm depName=sentences-per-line
	SentencesPerLineVersion = "0.5.2"
	// SetupBuildxActionVersion is the version of setup-buildx github action.
	// renovate: datasource=github-tags depName=docker/setup-buildx-action
	SetupBuildxActionVersion = "v4.0.0"
	SetupBuildxActionRef     = "4d04d5d9486b7bd6fa91e7baf45bbb4f8b9deedd"
	// SetupTerraformActionVersion is the version of setup terraform github action.
	// renovate: datasource=github-tags depName=hashicorp/setup-terraform
	SetupTerraformActionVersion = "v4.0.0"
	SetupTerraformActionRef     = "5e8dbf3c6d9deaf4193ca7a8fb23f2ac83bb6c85"
	// SlackNotifyActionVersion is the version of slack notify github action.
	// renovate: datasource=github-tags depName=slackapi/slack-github-action
	SlackNotifyActionVersion = "v2.1.1"
	SlackNotifyActionRef     = "91efab103c0de0a537f72a35f6b8cda0ee76bf0a"
	// SystemInfoActionVersion is the version of system info github action.
	// renovate: datasource=github-tags depName=kenchan0130/actions-system-info
	SystemInfoActionVersion = "v1.4.0"
	SystemInfoActionRef     = "59699597e84e80085a750998045983daa49274c4"
	// UploadArtifactActionVersion is the version of upload artifact github action.
	// renovate: datasource=github-tags depName=actions/upload-artifact
	UploadArtifactActionVersion = "v7.0.0"
	UploadArtifactActionRef     = "bbbca2ddaa5d8feaa63e36b76fdaad77386f024f"
	// VTProtobufVersion is the version of vtprotobuf.
	// renovate: datasource=go depName=github.com/planetscale/vtprotobuf
	VTProtobufVersion = "v0.6.0"
	// StaleActionVersion is the version of stale github action.
	// renovate: datasource=github-tags depName=actions/stale
	StaleActionVersion = "v10.2.0"
	StaleActionRef     = "b5d41d4e1d5dceea10e7104786b73624c18a190f"
	// LockThreadsActionVersion is the version of lock threads github action.
	// renovate: datasource=github-tags depName=dessant/lock-threads
	LockThreadsActionVersion = "v6.0.0"
	LockThreadsActionRef     = "7266a7ce5c1df01b1c6db85bf8cd86c737dadbe7"
)

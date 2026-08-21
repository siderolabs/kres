// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package config

const (
	// ContainerImageFrontendDockerfile is the dockerfile frontend.
	ContainerImageFrontendDockerfile = "Dockerfile"
	// ContainerImageFrontendPkgfile is the pkgfile frontend.
	ContainerImageFrontendPkgfile = "Pkgfile"

	// BldrImageVersion is the version of bldr image.
	// renovate: datasource=github-releases depName=siderolabs/bldr
	BldrImageVersion = "v0.6.3"

	// CheckOutActionVersion is the version of checkout github action.
	// renovate: datasource=github-tags depName=actions/checkout
	CheckOutActionVersion = "v7.0.0"
	CheckOutActionRef     = "9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0"
	// CodeCovActionVersion is the version of codecov github action.
	// renovate: datasource=github-tags depName=codecov/codecov-action
	CodeCovActionVersion = "v7.0.0"
	CodeCovActionRef     = "fb8b3582c8e4def4969c97caa2f19720cb33a72f"
	// DeepCopyVersion is the version of deepcopy.
	// renovate: datasource=go depName=github.com/siderolabs/deep-copy
	DeepCopyVersion = "v0.5.8"
	// DindContainerImageVersion is the version of the dind container image.
	// renovate: datasource=docker versioning=docker depName=docker
	DindContainerImageVersion = "29.6-dind"
	// DockerfileFrontendImageVersion is the version of the dockerfile frontend image.
	// renovate: datasource=docker versioning=docker depName=docker/dockerfile-upstream
	DockerfileFrontendImageVersion = "1.25.0-labs"
	// DownloadArtifactActionVersion is the version of download artifact github action.
	// renovate: datasource=github-tags depName=actions/download-artifact
	DownloadArtifactActionVersion = "v8.0.1"
	DownloadArtifactActionRef     = "3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c"
	// GitHubScriptActionVersion is the version of github script action.
	// renovate: datasource=github-tags depName=actions/github-script
	GitHubScriptActionVersion = "v9.0.0"
	GitHubScriptActionRef     = "3a2844b7e9c422d3c10d287c895573f7108da1b3"
	// GoFmtVersion is the version of gofmt.
	// renovate: datasource=go depName=github.com/mvdan/gofumpt
	GoFmtVersion = "v0.10.0"
	// GoImportsVersion is the version of goimports.
	// renovate: datasource=go depName=golang.org/x/tools
	GoImportsVersion = "v0.45.0"
	// GoMockVersion is the version of gomock.
	// renovate: datasource=go depName=github.com/uber-go/mock
	GoMockVersion = "v0.6.0"
	// GolangCIlintVersion is the version of golangci-lint.
	// renovate: datasource=go depName=github.com/golangci/golangci-lint
	GolangCIlintVersion = "v2.12.2"
	// DisVulnCheckVersion is the version of dis-vulncheck.
	// renovate: datasource=go versioning=loose depName=github.com/shanduur/dis-vulncheck
	DisVulnCheckVersion = "v0.0.0-20260430093434-b73e0972e2fb"
	// GoVersion is the version of Go.
	// renovate: datasource=github-tags extractVersion=^go(?<version>.*)$ depName=golang/go
	GoVersion = "1.26.7"
	// GrpcGatewayVersion is the version of grpc-gateway.
	// renovate: datasource=go depName=github.com/grpc-ecosystem/grpc-gateway
	GrpcGatewayVersion = "v2.29.0"
	// GrpcGoVersion is the version of grpc.
	// renovate: datasource=go depName=google.golang.org/grpc/cmd/protoc-gen-go-grpc
	GrpcGoVersion = "v1.6.2"
	// HelmUnitTestVersion is the version of helm unit test plugin.
	// renovate: datasource=github-tags depName=helm-unittest/helm-unittest
	HelmUnitTestVersion = "v1.1.1"
	// HelmValuesSchemaJSONVersion is the version of helm values-schema-json plugin.
	// renovate: datasource=github-tags depName=losisin/helm-values-schema-json
	HelmValuesSchemaJSONVersion = "v2.5.0"
	// HelmDocsVersion is the version of helm-docs tool.
	// renovate: datasource=github-tags depName=norwoodj/helm-docs
	HelmDocsVersion = "v1.14.2"
	// ImageSignerVersion is the version of the image-signer tool.
	// renovate: datasource=github-releases depName=siderolabs/go-tools
	ImageSignerVersion = "v0.3.2"
	// LoginActionVersion is the version of login github action.
	// renovate: datasource=github-tags depName=docker/login-action
	LoginActionVersion = "v4.4.0"
	LoginActionRef     = "af1e73f918a031802d376d3c8bbc3fe56130a9b0"
	// MarkdownLintCLIVersion is the version of markdownlint.
	// renovate: datasource=npm depName=markdownlint-cli
	MarkdownLintCLIVersion = "0.49.0"
	// BunContainerImageVersion is the default bun container image.
	// renovate: datasource=docker versioning=docker depName=oven/bun
	BunContainerImageVersion = "1.3.14-alpine"
	// NodeContainerImageVersion is the default node container image.
	//
	// NOTE: Check renovate.json for the rules on this before bumping, e.g., pinned versions.
	// As a rule of thumb, we bump only to the versions promoted to be LTS (even [not odd] major versions get promoted after a while, always check).
	//
	// renovate: datasource=docker versioning=docker depName=node
	NodeContainerImageVersion = "24.18.1-alpine"
	// PkgsVersion is the version of pkgs.
	// renovate: datasource=github-tags depName=siderolabs/pkgs
	PkgsVersion = "v1.13.0"
	// ProtobufGoVersion is the version of protobuf.
	// renovate: datasource=go depName=google.golang.org/protobuf/cmd/protoc-gen-go
	ProtobufGoVersion = "v1.36.11"
	// ProtobufTSGatewayVersion is the version of protoc-gen-grpc-gateway-ts.
	// renovate: datasource=go depName=github.com/siderolabs/protoc-gen-grpc-gateway-ts
	ProtobufTSGatewayVersion = "v1.4.1"
	// ReleaseActionVersion is the version of release github action.
	// renovate: datasource=github-tags depName=softprops/action-gh-release
	ReleaseActionVersion = "v3.0.1"
	ReleaseActionRef     = "718ea10b132b3b2eba29c1007bb80653f286566b"
	// SentencesPerLineVersion is the version of sentences-per-line.
	// renovate: datasource=npm depName=sentences-per-line
	SentencesPerLineVersion = "0.5.3"
	// SetupBuildxActionVersion is the version of setup-buildx github action.
	// renovate: datasource=github-tags depName=docker/setup-buildx-action
	SetupBuildxActionVersion = "v4.2.0"
	SetupBuildxActionRef     = "bb05f3f5519dd87d3ba754cc423b652a5edd6d2c"
	// SetupNodeActionVersion is the version of setup-node github action.
	// renovate: datasource=github-tags depName=actions/setup-node
	SetupNodeActionVersion = "v6.4.0"
	SetupNodeActionRef     = "48b55a011bda9f5d6aeb4c2d9c7362e8dae4041e"
	// ChromaticActionVersion is the version of the chromaui/action github action.
	// renovate: datasource=github-tags depName=chromaui/action
	ChromaticActionVersion = "v18.0.1"
	ChromaticActionRef     = "94713c544284a14195de3b50ef24301579f1877e"
	// SetupTerraformActionVersion is the version of setup terraform github action.
	// renovate: datasource=github-tags depName=hashicorp/setup-terraform
	SetupTerraformActionVersion = "v4.0.1"
	SetupTerraformActionRef     = "dfe3c3f87815947d99a8997f908cb6525fc44e9e"
	// SyftVersion is the version of syft used for SBOM generation.
	// renovate: datasource=go depName=github.com/anchore/syft
	SyftVersion = "v1.46.0"
	// SlackNotifyActionVersion is the version of slack notify github action.
	// renovate: datasource=github-tags depName=slackapi/slack-github-action
	SlackNotifyActionVersion = "v3.0.3"
	SlackNotifyActionRef     = "45a88b9581bfab2566dc881e2cd66d334e621e2c"
	// SystemInfoActionVersion is the version of system info github action.
	// renovate: datasource=github-tags depName=kenchan0130/actions-system-info
	SystemInfoActionVersion = "v1.4.0"
	SystemInfoActionRef     = "59699597e84e80085a750998045983daa49274c4"
	// UploadArtifactActionVersion is the version of upload artifact github action.
	// renovate: datasource=github-tags depName=actions/upload-artifact
	UploadArtifactActionVersion = "v7.0.1"
	UploadArtifactActionRef     = "043fb46d1a93c77aae656e7c1c64a875d1fc6a0a"
	// VTProtobufVersion is the version of vtprotobuf.
	// renovate: datasource=go depName=github.com/planetscale/vtprotobuf
	VTProtobufVersion = "v0.6.0"
	// StaleActionVersion is the version of stale github action.
	// renovate: datasource=github-tags depName=actions/stale
	StaleActionVersion = "v10.3.0"
	StaleActionRef     = "eb5cf3af3ac0a1aa4c9c45633dd1ae542a27a899"
	// LockThreadsActionVersion is the version of lock threads github action.
	// renovate: datasource=github-tags depName=dessant/lock-threads
	LockThreadsActionVersion = "v6.0.2"
	LockThreadsActionRef     = "89ae32b08ed1a541efecbab17912962a5e38981c"
)

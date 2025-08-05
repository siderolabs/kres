// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package ghworkflow_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/siderolabs/kres/internal/output"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
)

type GHWorkflowSuite struct {
	suite.Suite
}

func TestGHWorkflowSuite(t *testing.T) {
	suite.Run(t, new(GHWorkflowSuite))
}

func (suite *GHWorkflowSuite) SetupSuite() {
	output.PreambleTimestamp, _ = time.Parse(time.RFC3339, strings.ReplaceAll(time.RFC3339, "07:00", "")) //nolint:errcheck
	output.PreambleCreator = "test"
}

func (suite *GHWorkflowSuite) TestDefaultWorkflows() {
	var (
		defaultBranch      = "main"
		withDefaultJob     = true
		withStaleJob       = true
		customSlackChannel = "ci-failure-custom"
	)

	output := ghworkflow.NewOutput(defaultBranch, withDefaultJob, withStaleJob, customSlackChannel) //nolint:typecheck

	var buf bytes.Buffer

	err := output.GenerateFile(ghworkflow.CiWorkflow, &buf)
	suite.Require().NoError(err)

	suite.Equal(`# THIS FILE WAS AUTOMATICALLY GENERATED, PLEASE DO NOT EDIT.
#
# Generated on 2006-01-02T15:04:05Z by test.

concurrency:
  group: ${{ github.head_ref || github.run_id }}
  cancel-in-progress: true
"on":
  push:
    branches:
      - main
      - release-*
    tags:
      - v*
  pull_request:
    branches:
      - main
      - release-*
name: default
jobs:
  default:
    permissions:
      actions: read
      contents: write
      issues: read
      packages: write
      pull-requests: read
    runs-on: []
    if: (!startsWith(github.head_ref, 'renovate/') && !startsWith(github.head_ref, 'dependabot/'))
    steps:
      - name: gather-system-info
        id: system-info
        uses: kenchan0130/actions-system-info@v1.3.1
        continue-on-error: true
      - name: print-system-info
        run: |
          MEMORY_GB=$((${{ steps.system-info.outputs.totalmem }}/1024/1024/1024))

          OUTPUTS=(
            "CPU Core: ${{ steps.system-info.outputs.cpu-core }}"
            "CPU Model: ${{ steps.system-info.outputs.cpu-model }}"
            "Hostname: ${{ steps.system-info.outputs.hostname }}"
            "NodeName: ${NODE_NAME}"
            "Kernel release: ${{ steps.system-info.outputs.kernel-release }}"
            "Kernel version: ${{ steps.system-info.outputs.kernel-version }}"
            "Name: ${{ steps.system-info.outputs.name }}"
            "Platform: ${{ steps.system-info.outputs.platform }}"
            "Release: ${{ steps.system-info.outputs.release }}"
            "Total memory: ${MEMORY_GB} GB"
          )

          for OUTPUT in "${OUTPUTS[@]}";do
            echo "${OUTPUT}"
          done
        continue-on-error: true
      - name: checkout
        uses: actions/checkout@v4
      - name: Unshallow
        run: |
          git fetch --prune --unshallow
      - name: Set up Docker Buildx
        id: setup-buildx
        uses: docker/setup-buildx-action@v3
        with:
          driver: remote
          endpoint: tcp://buildkit-amd64.ci.svc.cluster.local:1234
        timeout-minutes: 10`,
		strings.TrimSpace(buf.String()))

	var buf2 bytes.Buffer

	err = output.GenerateFile(ghworkflow.SlackCIFailureWorkflow, &buf2)

	fmt.Print(buf2.String())
	suite.Require().NoError(err)
	suite.Contains(buf2.String(), customSlackChannel)
}

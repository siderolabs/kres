// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package ghworkflow_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/siderolabs/kres/internal/output"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
)

// set to true to regenerate testdata golden files.
var update = false //nolint:gochecknoglobals

func assertGolden(t testing.TB, filename string, got []byte) {
	t.Helper()

	path := filepath.Join("testdata", filepath.FromSlash(t.Name()), filepath.Base(filename))

	if update {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, got, 0o600))

		return
	}

	want, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, string(want), string(got))
}

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

	o := ghworkflow.NewOutput(defaultBranch, withDefaultJob, withStaleJob, customSlackChannel) //nolint:typecheck
	o.SetRunnerGroup(ghworkflow.GenericRunner)

	var ciBuf bytes.Buffer

	suite.Require().NoError(o.GenerateFile(ghworkflow.CiWorkflow, &ciBuf))
	assertGolden(suite.T(), ghworkflow.CiWorkflow, ciBuf.Bytes())

	var slackBuf bytes.Buffer

	suite.Require().NoError(o.GenerateFile(ghworkflow.SlackCIFailureWorkflow, &slackBuf))
	assertGolden(suite.T(), ghworkflow.SlackCIFailureWorkflow, slackBuf.Bytes())
}

func TestMatrixStrategy(t *testing.T) {
	output.PreambleTimestamp, _ = time.Parse(time.RFC3339, strings.ReplaceAll(time.RFC3339, "07:00", "")) //nolint:errcheck
	output.PreambleCreator = "test"

	// Build a step that has a matrix condition and a matrix-interpolated command.
	buildStep := ghworkflow.Step("build").
		SetMakeStep("build-${{ matrix.track }}")

	err := buildStep.SetConditions("only-on-schedule", "matrix.buildEnforcing")
	require.NoError(t, err)

	// Build a triggered workflow job with a 2-entry matrix.
	job := &ghworkflow.Job{
		If:     "github.event.workflow_run.conclusion == 'success'",
		RunsOn: ghworkflow.NewRunsOnGroupLabel("large", ""),
		Strategy: &ghworkflow.Strategy{
			MaxParallel: 2,
			Matrix: &ghworkflow.StrategyMatrix{
				Include: []map[string]string{
					{"track": "0"},
					{"track": "1", "buildEnforcing": "true"},
				},
			},
		},
		Steps: []*ghworkflow.JobStep{buildStep},
	}

	o := ghworkflow.NewOutput("main", false, false, "")
	o.AddWorkflow("integration-provision-triggered", &ghworkflow.Workflow{
		Name: "integration-provision-triggered",
		Concurrency: ghworkflow.Concurrency{
			Group:            "${{ github.head_ref || github.run_id }}",
			CancelInProgress: true,
		},
		On: ghworkflow.On{
			WorkFlowRun: ghworkflow.WorkFlowRun{
				Workflows: []string{"artifacts-cron"},
				Types:     []string{"completed"},
			},
		},
		Jobs: map[string]*ghworkflow.Job{
			ghworkflow.DefaultJobName: job,
		},
	})

	var buf bytes.Buffer

	require.NoError(t, o.GenerateFile(".github/workflows/integration-provision-triggered.yaml", &buf))
	assertGolden(t, "integration-provision-triggered.yaml", buf.Bytes())
}

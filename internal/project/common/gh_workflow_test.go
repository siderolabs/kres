// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/siderolabs/kres/internal/output"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/project/common"
	"github.com/siderolabs/kres/internal/project/meta"
)

// set to true to regenerate testdata golden files.
var update = false //nolint:gochecknoglobals

func assertGolden(t *testing.T, filename string, got []byte) {
	t.Helper()

	path := filepath.Join("testdata", t.Name(), filepath.Base(filename))

	if update {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, got, 0o600))

		return
	}

	want, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, string(want), string(got))
}

func newTestOutput() *ghworkflow.Output {
	o := ghworkflow.NewOutput("main", true, false, "")
	o.SetRunnerGroup(ghworkflow.GenericRunner)

	return o
}

func compileWorkflow(t *testing.T, jobs []common.Job) *ghworkflow.Output {
	t.Helper()

	output.PreambleTimestamp, _ = time.Parse(time.RFC3339, strings.ReplaceAll(time.RFC3339, "07:00", "")) //nolint:errcheck
	output.PreambleCreator = "test"

	m := &meta.Options{CompileGithubWorkflowsOnly: true}
	gw := common.NewGHWorkflow(m)
	gw.Jobs = jobs

	o := newTestOutput()

	require.NoError(t, gw.CompileGitHubWorkflow(o))

	return o
}

// TestMatrixLabelKeysExpansion verifies that when a matrix job has LabelKeys set,
// CompileGitHubWorkflow emits N flat jobs in ci.yaml instead of a single matrix job.
func TestMatrixLabelKeysExpansion(t *testing.T) {
	jobs := []common.Job{
		{
			Name:        "integration-aws-nvidia-nonfree",
			RunnerGroup: "large",
			Depends:     []string{"default"},
			TriggerLabels: []string{
				"integration/aws-nvidia-nonfree",
				"integration/aws-nvidia",
			},
			Matrix: &common.Matrix{
				MaxParallel: 2,
				LabelKeys:   []string{"variant", "arch"},
				Include: []common.MatrixInclude{
					{Values: common.MatrixEntry{"variant": "lts", "arch": "amd64"}},
					{Values: common.MatrixEntry{"variant": "production", "arch": "arm64"}},
				},
			},
			Steps: []common.Step{
				{
					Name:    "run-tests",
					Command: "integration-${{ matrix.variant }}-${{ matrix.arch }}",
					Environment: map[string]string{
						"VARIANT": "${{ matrix.variant }}",
						"ARCH":    "${{ matrix.arch }}",
					},
				},
				{
					Name: "save-logs",
					ArtifactStep: &common.ArtifactStep{
						Type:                            "upload",
						ArtifactName:                    "logs-${{ matrix.variant }}-${{ matrix.arch }}",
						ArtifactPath:                    "/tmp/logs",
						DisableExecutableListGeneration: true,
					},
				},
			},
		},
	}

	o := compileWorkflow(t, jobs)

	var buf bytes.Buffer

	require.NoError(t, o.GenerateFile(ghworkflow.CiWorkflow, &buf))
	assertGolden(t, "ci.yaml", buf.Bytes())
}

// TestMatrixConditionResolution verifies that matrix.* conditions are resolved at
// compile time for flat jobs: steps whose matrix condition is not satisfied by an
// entry are dropped; steps whose condition is satisfied have no if-guard for it.
func TestMatrixConditionResolution(t *testing.T) {
	jobs := []common.Job{
		{
			Name:        "integration-provision",
			RunnerGroup: "large",
			Depends:     []string{"default"},
			TriggerLabels: []string{
				"integration/provision",
			},
			Matrix: &common.Matrix{
				MaxParallel: 2,
				LabelKeys:   []string{"track"},
				Include: []common.MatrixInclude{
					{Values: common.MatrixEntry{"track": "0"}},
					{Values: common.MatrixEntry{"track": "1", "buildEnforcing": "true"}},
				},
			},
			Steps: []common.Step{
				{
					Name:    "always-step",
					Command: "always-${{ matrix.track }}",
				},
				{
					Name:    "enforcing-only",
					Command: "enforcing",
					Conditions: []string{
						"only-on-schedule",
						"matrix.buildEnforcing",
					},
				},
			},
		},
	}

	o := compileWorkflow(t, jobs)

	var buf bytes.Buffer

	require.NoError(t, o.GenerateFile(ghworkflow.CiWorkflow, &buf))
	assertGolden(t, "ci.yaml", buf.Bytes())
}

// TestTriggeredRunIDAutoInject verifies that a triggered workflow without an
// explicit RunID on the download step gets run-id auto-injected, while the
// ci.yaml job for the same job definition does NOT get run-id.
func TestTriggeredRunIDAutoInject(t *testing.T) {
	jobs := []common.Job{
		{
			Name:        "integration-qemu-race",
			RunnerGroup: "large",
			Depends:     []string{"default"},
			TriggerLabels: []string{
				"integration/qemu-race",
			},
			OnWorkflowRun: &common.OnWorkflowRun{
				Workflows: []string{"artifacts-cron"},
				Types:     []string{"completed"},
			},
			Steps: []common.Step{
				{
					Name: "download-artifacts",
					ArtifactStep: &common.ArtifactStep{
						Type:                            "download",
						ArtifactName:                    "talos-artifacts",
						ArtifactPath:                    "_out",
						DisableExecutableListGeneration: true,
					},
				},
			},
		},
	}

	o := compileWorkflow(t, jobs)

	var triggeredBuf bytes.Buffer

	require.NoError(t, o.GenerateFile(".github/workflows/integration-qemu-race-triggered.yaml", &triggeredBuf))
	assertGolden(t, "integration-qemu-race-triggered.yaml", triggeredBuf.Bytes())

	var ciBuf bytes.Buffer

	require.NoError(t, o.GenerateFile(ghworkflow.CiWorkflow, &ciBuf))
	assertGolden(t, "ci.yaml", ciBuf.Bytes())
}

// TestArtifactStepRunID verifies that RunID on ArtifactStep is rendered as
// run-id in the download step, and that the triggered-workflow auto-inject
// does not overwrite it.
func TestArtifactStepRunID(t *testing.T) {
	jobs := []common.Job{
		{
			Name:        "integration-provision",
			RunnerGroup: "large",
			OnWorkflowRun: &common.OnWorkflowRun{
				Workflows: []string{"artifacts-cron"},
				Types:     []string{"completed"},
			},
			Steps: []common.Step{
				{
					Name: "download-artifacts",
					ArtifactStep: &common.ArtifactStep{
						Type:                            "download",
						ArtifactName:                    "talos-artifacts",
						ArtifactPath:                    "_out",
						RunID:                           "${{ github.event.workflow_run.id || github.run_id }}",
						DisableExecutableListGeneration: true,
					},
				},
			},
		},
	}

	o := compileWorkflow(t, jobs)

	var buf bytes.Buffer

	require.NoError(t, o.GenerateFile(".github/workflows/integration-provision-triggered.yaml", &buf))
	assertGolden(t, "integration-provision-triggered.yaml", buf.Bytes())
}

// TestTriggeredWorkflowMatrixFailFast verifies that a triggered workflow with a
// matrix strategy has fail-fast: false so that one entry failing does not cancel
// the remaining entries.
func TestTriggeredWorkflowMatrixFailFast(t *testing.T) {
	jobs := []common.Job{
		{
			Name:        "integration-misc-3",
			RunnerGroup: "large",
			Depends:     []string{"default"},
			TriggerLabels: []string{
				"integration/misc-3",
			},
			OnWorkflowRun: &common.OnWorkflowRun{
				Workflows: []string{"artifacts-cron"},
				Types:     []string{"completed"},
			},
			Matrix: &common.Matrix{
				MaxParallel: 1,
				LabelKeys:   []string{"variant"},
				Include: []common.MatrixInclude{
					{Values: common.MatrixEntry{"variant": "default"}},
					{Values: common.MatrixEntry{"variant": "enforcing", "buildEnforcing": "true"}},
				},
			},
			Steps: []common.Step{
				{Name: "run-tests", Command: "e2e-${{ matrix.variant }}"},
			},
		},
	}

	o := compileWorkflow(t, jobs)

	var buf bytes.Buffer

	require.NoError(t, o.GenerateFile(".github/workflows/integration-misc-3-triggered.yaml", &buf))
	assertGolden(t, "integration-misc-3-triggered.yaml", buf.Bytes())
}

// TestFlatJobMatrix verifies that a job with flatJobMatrix: true emits a matrix
// strategy in ci.yaml (with max-parallel and fail-fast: false) instead of flat
// job expansion, and that the name field is set from labelKeys.
func TestFlatJobMatrix(t *testing.T) {
	jobs := []common.Job{
		{
			Name:        "integration-misc-0",
			RunnerGroup: "large",
			Depends:     []string{"default"},
			TriggerLabels: []string{
				"integration/misc-0",
			},
			Matrix: &common.Matrix{
				MaxParallel:   2,
				FlatJobMatrix: true,
				LabelKeys:     []string{"test"},
				Include: []common.MatrixInclude{
					{Values: common.MatrixEntry{"test": "e2e-firewall", "shortIntegrationTest": "yes"}},
					{Values: common.MatrixEntry{"test": "e2e-canal-reset", "integrationTestRun": "TestIntegration/api.ResetSuite/TestResetWithSpec"}},
					{Values: common.MatrixEntry{"test": "e2e-controlplane-port", "shortIntegrationTest": "yes", "withControlPlanePort": "443"}},
				},
			},
			Steps: []common.Step{
				{
					Name:     "e2e-qemu",
					Command:  "e2e-qemu",
					WithSudo: true,
					Environment: map[string]string{
						"SHORT_INTEGRATION_TEST":  "${{ matrix.shortIntegrationTest }}",
						"INTEGRATION_TEST_RUN":    "${{ matrix.integrationTestRun }}",
						"WITH_CONTROL_PLANE_PORT": "${{ matrix.withControlPlanePort }}",
					},
				},
			},
		},
	}

	o := compileWorkflow(t, jobs)

	var buf bytes.Buffer

	require.NoError(t, o.GenerateFile(ghworkflow.CiWorkflow, &buf))
	assertGolden(t, "ci.yaml", buf.Bytes())
}

// TestMatrixPerEntryTriggerLabels verifies that per-entry TriggerLabels fire only
// the matching flat job, while job-level TriggerLabels fire all flat jobs.
// This tests the combined oss/nonfree scenario where integration/aws-nvidia-oss must
// NOT trigger nonfree flat jobs.
func TestMatrixPerEntryTriggerLabels(t *testing.T) {
	jobs := []common.Job{
		{
			Name:        "integration-aws-nvidia",
			RunnerGroup: "large",
			Depends:     []string{"default"},
			TriggerLabels: []string{
				"integration/aws-nvidia", // job-level: fires all flat jobs
			},
			Matrix: &common.Matrix{
				MaxParallel: 2,
				LabelKeys:   []string{"driver", "variant"},
				Include: []common.MatrixInclude{
					{
						Values:        common.MatrixEntry{"driver": "oss", "variant": "lts"},
						TriggerLabels: []string{"integration/aws-nvidia-oss"},
					},
					{
						Values:        common.MatrixEntry{"driver": "nonfree", "variant": "lts"},
						TriggerLabels: []string{"integration/aws-nvidia-nonfree"},
					},
				},
			},
			Steps: []common.Step{
				{Name: "run-tests", Command: "e2e-${{ matrix.driver }}-${{ matrix.variant }}"},
			},
		},
	}

	o := compileWorkflow(t, jobs)

	var buf bytes.Buffer

	require.NoError(t, o.GenerateFile(ghworkflow.CiWorkflow, &buf))
	assertGolden(t, "ci.yaml", buf.Bytes())
}

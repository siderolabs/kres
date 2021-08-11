// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package dockerfile_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/talos-systems/kres/internal/output"
	"github.com/talos-systems/kres/internal/output/dockerfile"
	"github.com/talos-systems/kres/internal/output/dockerfile/step"
)

type DockerfileSuite struct {
	suite.Suite
}

func (suite *DockerfileSuite) SetupSuite() {
	output.PreambleTimestamp, _ = time.Parse(time.RFC3339, strings.ReplaceAll(time.RFC3339, "07:00", "")) //nolint:errcheck
	output.PreambleCreator = "test"
}

func (suite *DockerfileSuite) TestGenerateFile() {
	output := &dockerfile.Output{}

	output.Stage("build").From("setup").Step(step.WorkDir("/src"))

	output.Stage("foo").From("bar")

	output.Stage("setup").From("scratch").Description("initialize tools").
		Step(step.Copy("src", "/workdir/src").From("source")).
		Step(step.Copy(".", "."))

	var buf bytes.Buffer

	err := output.GenerateFile("Dockerfile", &buf)
	suite.Require().NoError(err)

	suite.Assert().Equal(`# syntax = docker/dockerfile-upstream:1.2.0-labs

# THIS FILE WAS AUTOMATICALLY GENERATED, PLEASE DO NOT EDIT.
#
# Generated on 2006-01-02T15:04:05Z by test.


FROM bar AS foo

# initialize tools
FROM scratch AS setup
COPY --from=source src /workdir/src
COPY . .

FROM setup AS build
WORKDIR /src

`, buf.String())
}

func TestDockerfileSuite(t *testing.T) {
	suite.Run(t, new(DockerfileSuite))
}

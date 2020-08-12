// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package makefile_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/talos-systems/kres/internal/output"
	"github.com/talos-systems/kres/internal/output/makefile"
)

type MakefileSuite struct {
	suite.Suite
}

func (suite *MakefileSuite) SetupSuite() {
	output.PreambleTimestamp, _ = time.Parse(time.RFC3339, strings.ReplaceAll(time.RFC3339, "07:00", "")) //nolint: errcheck
	output.PreambleCreator = "test"
}

func (suite *MakefileSuite) TestGenerateFile() {
	output := &makefile.Output{}

	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.SimpleVariable("FOO", "bar")).
		Variable(makefile.RecursiveVariable("BLA", "bla")).
		Variable(makefile.OverridableVariable("DEFAULT", "unknown"))

	output.VariableGroup(makefile.VariableGroupDocker).
		Variable(makefile.SimpleVariable("BUILD", "docker buildx build")).
		Variable(makefile.RecursiveVariable("ARGS", "do it").Push("once").Push("more")).
		Variable(makefile.OverridableVariable("CI_ARGS", ""))

	output.Target("all").
		Depends("foo", "bar")

	output.Target("build").
		Phony().
		Description("build everything").
		Script("cc -o a.out", "ld a.out\nar a.out")

	var buf bytes.Buffer

	err := output.GenerateFile("Makefile", &buf)
	suite.Require().NoError(err)

	suite.Assert().Equal(`# THIS FILE WAS AUTOMATICALLY GENERATED, PLEASE DO NOT EDIT.
#
# Generated on 2006-01-02T15:04:05Z by test.

# common variables

FOO := bar
BLA = bla
DEFAULT ?= unknown

# docker build settings

BUILD := docker buildx build
ARGS = do it
ARGS += once
ARGS += more
CI_ARGS ?=

all: foo bar

.PHONY: build
build:  ## build everything
	cc -o a.out
	ld a.out
	ar a.out

`, buf.String())
}

func TestMakefileSuite(t *testing.T) {
	suite.Run(t, new(MakefileSuite))
}

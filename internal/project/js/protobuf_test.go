// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package js_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/lefthook"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/js"
	"github.com/siderolabs/kres/internal/project/meta"
)

func TestProtobufInterfaces(t *testing.T) {
	assert.Implements(t, (*dockerfile.Compiler)(nil), new(js.Protobuf))
	assert.Implements(t, (*makefile.Compiler)(nil), new(js.Protobuf))
	assert.Implements(t, (*lefthook.Compiler)(nil), new(js.Protobuf))
}

func TestProtobufLefthook(t *testing.T) {
	proto := js.NewProtobuf(&meta.Options{}, "frontend")

	// `make generate-frontend` joins the shared fix stage as a named job.
	output := lefthook.NewOutput()
	output.Enable()

	require.NoError(t, proto.CompileLefthook(output))

	var buf bytes.Buffer

	require.NoError(t, output.GenerateFile("lefthook.yml", &buf))

	rendered := buf.String()
	assert.Contains(t, rendered, "name: generate frontend")
	assert.Contains(t, rendered, "run: make generate-frontend")
	assert.Contains(t, rendered, "stage_fixed: true")
}

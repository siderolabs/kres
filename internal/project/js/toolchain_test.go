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
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/output/template"
	"github.com/siderolabs/kres/internal/project/js"
	"github.com/siderolabs/kres/internal/project/meta"
)

func TestToolchainInterfaces(t *testing.T) {
	assert.Implements(t, (*dockerfile.Compiler)(nil), new(js.Toolchain))
	assert.Implements(t, (*template.Compiler)(nil), new(js.Toolchain))
	assert.Implements(t, (*makefile.Compiler)(nil), new(js.Toolchain))
	assert.Implements(t, (*ghworkflow.Compiler)(nil), new(js.Toolchain))
	assert.Implements(t, (*makefile.SkipAsMakefileDependency)(nil), new(js.Toolchain))
}

func TestToolchainPackages(t *testing.T) {
	for _, test := range []struct {
		name          string
		want          string
		extraPackages []string
		protobuf      bool
	}{
		{
			name: "none",
		},
		{
			name:          "extra",
			extraPackages: []string{"git", "chromium"},
			want:          "RUN apk --update --no-cache add git chromium\n",
		},
		{
			name:          "protobuf",
			extraPackages: []string{"git", "go"},
			protobuf:      true,
			want:          "RUN apk --update --no-cache add go protoc protobuf-dev git\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			options := &meta.Options{}

			toolchain := js.NewToolchain(options, "frontend")
			toolchain.ExtraPackages = test.extraPackages

			if test.protobuf {
				protobuf := js.NewProtobuf(options, "frontend")
				protobuf.Specs = []js.ProtoSpec{{Source: "api/service.proto"}}

				toolchain.AddInput(protobuf)
			}

			output := dockerfile.NewOutput()
			require.NoError(t, toolchain.CompileDockerfile(output))

			var buf bytes.Buffer

			require.NoError(t, output.GenerateFile("Dockerfile", &buf))

			if test.want == "" {
				assert.NotContains(t, buf.String(), "apk")

				return
			}

			assert.Contains(t, buf.String(), test.want)
		})
	}
}

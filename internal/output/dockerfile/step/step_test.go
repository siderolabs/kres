// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package step_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/siderolabs/kres/internal/output/dockerfile/step"
)

func TestGenerate(t *testing.T) {
	for _, tt := range []struct {
		step     step.Step
		expected string
	}{
		{
			step.Copy("/src", "/dst"),
			"COPY /src /dst\n",
		},
		{
			step.Copy("/src", "/dst").From("somestage"),
			"COPY --from=somestage /src /dst\n",
		},
		{
			step.Env("GO111MODULE", "on"),
			"ENV GO111MODULE=on\n",
		},
		{
			step.Run("go", "build", "./..."),
			"RUN go build ./...\n",
		},
		{
			step.Run("go", "build", "-ldflags", "-s -x -W", "./..."),
			"RUN go build -ldflags '-s -x -W' ./...\n",
		},
		{
			step.Run("go", "build", "./...").SecurityInsecure(),
			"RUN --security=insecure go build ./...\n",
		},
		{
			step.Run("go", "build", "./...").SecurityInsecure().Env("CGOENABLED", "0").Env("BUILD", "1"),
			"RUN --security=insecure CGOENABLED=0 BUILD=1 go build ./...\n",
		},
		{
			step.Run("go", "build", "./...").MountCache("/root/go/.cache", "prj"),
			"RUN --mount=type=cache,target=/root/go/.cache,id=prj/root/go/.cache go build ./...\n",
		},
		{
			step.Run("go", "build", "./...").MountCache("/root/go/.cache", "prj"),
			"RUN --mount=type=cache,target=/root/go/.cache,id=prj/root/go/.cache go build ./...\n",
		},
		{
			step.Run("go", "build", "./...").MountCache("/root/go/.cache", "prj", step.CacheLocked),
			"RUN --mount=type=cache,target=/root/go/.cache,id=prj/root/go/.cache,sharing=locked go build ./...\n",
		},
		{
			step.Script("curl http://example.com/ | tar xzf -").MountCache("/root/go/.cache", "prj"),
			"RUN --mount=type=cache,target=/root/go/.cache,id=prj/root/go/.cache curl http://example.com/ | tar xzf -\n",
		},
		{
			step.Arg("GOFUMPT_VERSION"),
			"ARG GOFUMPT_VERSION\n",
		},
		{
			step.WorkDir("/src"),
			"WORKDIR /src\n",
		},
		{
			step.Entrypoint("/bldr", "frontend"),
			"ENTRYPOINT [\"/bldr\",\"frontend\"]\n",
		},
		{
			step.Add("./src", "/dst"),
			"ADD ./src /dst\n",
		},
		{
			step.Label("foo", "bar"),
			"LABEL foo=bar\n",
		},
	} {
		var buf bytes.Buffer

		require.NoError(t, tt.step.Generate(&buf))

		assert.Equal(t, tt.expected, buf.String())
	}
}

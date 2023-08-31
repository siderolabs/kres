// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	_ "embed"

	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/project/meta"
)

//nolint:unused
//go:embed files/slack-notify-payload.json
var slackNotifyPayloadJSON string

// SlackNotify provides common Slack notification target.
type SlackNotify struct {
	meta *meta.Options
	dag.BaseNode
}

// NewSlackNotify initializes SlackNotify.
func NewSlackNotify(meta *meta.Options) *SlackNotify {
	return &SlackNotify{
		meta: meta,
	}
}

// CompileGitHubWorkflow implements ghworkflow.Compiler.
func (slacknotify *SlackNotify) CompileGitHubWorkflow(_ *ghworkflow.Output) error {
	// TODO: enable once we figure out secrets for forks.
	// output.AddStep(
	// 	"default",
	// 	&ghworkflow.Step{
	// 		Name: "Slack Notify",
	// 		If:   "always()",
	// 		Uses: fmt.Sprintf("slackapi/slack-github-action@%s", config.SlackNotifyActionVersion),
	// 		Env: map[string]string{
	// 			"SLACK_BOT_TOKEN": "${{ secrets.SLACK_BOT_TOKEN }}",
	// 		},
	// 		With: map[string]string{
	// 			"channel-id": "proj-talos-maintainers",
	// 			"payload":    slackNotifyPayloadJSON,
	// 		},
	// 	},
	// )
	return nil
}

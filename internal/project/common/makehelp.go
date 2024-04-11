// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// MakeHelp provides Makefile `help` target.
type MakeHelp struct {
	dag.BaseNode

	meta *meta.Options

	MenuHeader string `yaml:"menuHeader"`
}

// NewMakeHelp initializes Build.
func NewMakeHelp(meta *meta.Options) *MakeHelp {
	return &MakeHelp{
		BaseNode: dag.NewBaseNode("build"),

		meta: meta,

		MenuHeader: defaultMenuHeader,
	}
}

// CompileMakefile implements makefile.Compiler.
func (help *MakeHelp) CompileMakefile(output *makefile.Output) error {
	output.VariableGroup(makefile.VariableGroupHelp).
		Variable(makefile.MultilineVariable("HELP_MENU_HEADER", help.MenuHeader).Export())

	output.Target("help").
		Description("This help menu.").
		Script("@echo \"$$HELP_MENU_HEADER\"").
		Script(`@grep -E '^[a-zA-Z%_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'`).
		Phony()

	return nil
}

const defaultMenuHeader = `# Getting Started

To build this project, you must have the following installed:

- git
- make
- docker (19.03 or higher)

## Creating a Builder Instance

The build process makes use of experimental Docker features (buildx).
To enable experimental features, add 'experimental: "true"' to '/etc/docker/daemon.json' on
Linux or enable experimental features in Docker GUI for Windows or Mac.

To create a builder instance, run:

	docker buildx create --name local --use

If running builds that needs to be cached aggresively create a builder instance with the following:

	docker buildx create --name local --use --config=config.toml

config.toml contents:

[worker.oci]
  gc = true
  gckeepstorage = 50000

  [[worker.oci.gcpolicy]]
    keepBytes = 10737418240
    keepDuration = 604800
    filters = [ "type==source.local", "type==exec.cachemount", "type==source.git.checkout"]
  [[worker.oci.gcpolicy]]
    all = true
    keepBytes = 53687091200

If you already have a compatible builder instance, you may use that instead.

## Artifacts

All artifacts will be output to ./$(ARTIFACTS). Images will be tagged with the
registry "$(REGISTRY)", username "$(USERNAME)", and a dynamic tag (e.g. $(IMAGE):$(IMAGE_TAG)).
The registry and username can be overridden by exporting REGISTRY, and USERNAME
respectively.
`

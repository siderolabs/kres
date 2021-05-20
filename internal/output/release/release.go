// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package release implements output for releases.
package release

import (
	"errors"
	"fmt"
	"io"
	"os"
	"text/template"

	"github.com/talos-systems/kres/internal/output"
	"github.com/talos-systems/kres/internal/project/meta"
)

const (
	release           = "./hack/release.sh"
	releaseTemplate   = "./hack/release.toml"
	config            = "./hack/git-chglog/config.yaml"
	changelogTemplate = "./hack/git-chglog/CHANGELOG.tpl.md"
)

const releaseStr = `
#!/bin/bash

set -e

RELEASE_TOOL_IMAGE="ghcr.io/talos-systems/release-tool:latest"

function release-tool {
  docker pull "${RELEASE_TOOL_IMAGE}" >/dev/null
  docker run --rm -w /src -v "${PWD}":/src:ro "${RELEASE_TOOL_IMAGE}" -l -d -n -t "${1}" ./hack/release.toml
}

function changelog {
  if [ "$#" -eq 1 ]; then
    (release-tool ${1}; echo; cat CHANGELOG.md) > CHANGELOG.md- && mv CHANGELOG.md- CHANGELOG.md
  else
    echo 1>&2 "Usage: $0 changelog [tag]"
    exit 1
  fi
}

function release-notes {
  release-tool "${2}" > "${1}"
}

function cherry-pick {
  if [ $# -ne 2 ]; then
    echo 1>&2 "Usage: $0 cherry-pick <commit> <branch>"
    exit 1
  fi

  git checkout $2
  git fetch
  git rebase upstream/$2
  git cherry-pick -x $1
}

function commit {
  if [ $# -ne 1 ]; then
    echo 1>&2 "Usage: $0 commit <tag>"
    exit 1
  fi

  git commit -s -m "release($1): prepare release" -m "This is the official $1 release."
}

if declare -f "$1" > /dev/null
then
  cmd="$1"
  shift
  $cmd "$@"
else
  cat <<EOF
Usage:
  commit:        Create the official release commit message.
  cherry-pick:   Cherry-pick a commit into a release branch.
  changelog:     Update the specified CHANGELOG.
  release-notes: Create release notes for GitHub release.
EOF

  exit 1
fi`

const configStr = `style: github
template: CHANGELOG.tpl.md
info:
  title: CHANGELOG
  repository_url: https://github.com/{{ .GitHubOrganization }}/{{ .GitHubRepository }}
options:
  commits:
    # filters:
    #   Type:
    #     - feat
    #     - fix
    #     - perf
    #     - refactor
  commit_groups:
    # title_maps:
    #   feat: Features
    #   fix: Bug Fixes
    #   perf: Performance Improvements
    #   refactor: Code Refactoring
  header:
    pattern: "^(\\w*)(?:\\(([\\w\\$\\.\\-\\*\\s]*)\\))?\\:\\s(.*)$"
    pattern_maps:
      - Type
      - Scope
      - Subject
  notes:
    keywords:
      - BREAKING CHANGE
`

const releaseTemplateStr = `
# commit to be tagged for the new release
commit = "HEAD"

project_name = "{{ .GitHubRepository }}"
github_repo = "{{ .GitHubOrganization}}/{{ .GitHubRepository }}"
match_deps = "^github.com/({{ .GitHubOrganization }}/[a-zA-Z0-9-]+)$"

# previous = -
# pre_release = true

# [notes]
`

const templateStr = `{{ range .Versions }}
<a name="{{ .Tag.Name }}"></a>
## {{ if .Tag.Previous }}[{{ .Tag.Name }}]({{ $.Info.RepositoryURL }}/compare/{{ .Tag.Previous.Name }}...{{ .Tag.Name }}){{ else }}{{ .Tag.Name }}{{ end }} ({{ datetime "2006-01-02" .Tag.Date }})

{{ range .CommitGroups -}}
### {{ .Title }}

{{ range .Commits -}}
* {{ if .Scope }}**{{ .Scope }}:** {{ end }}{{ .Subject }}
{{ end }}
{{ end -}}

{{- if .NoteGroups -}}
{{ range .NoteGroups -}}
### {{ .Title }}

{{ range .Notes }}
{{ .Body }}
{{ end }}
{{ end -}}
{{ end -}}
{{ end -}}`

// Output implements .gitignore generation.
type Output struct {
	output.FileAdapter

	meta *meta.Options
}

// NewOutput creates new Makefile output.
func NewOutput() *Output {
	output := &Output{}

	output.FileAdapter.FileWriter = output

	return output
}

// Compile implements output.Writer interface.
func (o *Output) Compile(node interface{}) error {
	compiler, implements := node.(Compiler)

	if !implements {
		return nil
	}

	return compiler.CompileRelease(o)
}

// Filenames implements output.FileWriter interface.
func (o *Output) Filenames() []string {
	return []string{release, releaseTemplate, config, changelogTemplate}
}

// SetMeta grabs build options.
func (o *Output) SetMeta(meta *meta.Options) {
	o.meta = meta
}

// GenerateFile implements output.FileWriter interface.
func (o *Output) GenerateFile(filename string, w io.Writer) error {
	switch filename {
	case release:
		return o.release(w)
	case releaseTemplate:
		return o.releaseTemplate(filename, w)
	case config:
		return o.config(w)
	case changelogTemplate:
		return o.changelogTemplate(w)
	default:
		panic("unexpected filename: " + filename)
	}
}

// Permissions implements output.PermissionsWriter interface.
func (o *Output) Permissions(filename string) os.FileMode {
	if filename == release {
		return 0o744
	}

	return 0
}

func (o *Output) release(w io.Writer) error {
	if _, err := w.Write([]byte("#!/bin/bash\n\n")); err != nil {
		return err
	}

	if _, err := w.Write([]byte(output.Preamble("# "))); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%s\n", releaseStr); err != nil {
		return err
	}

	return nil
}

func (o *Output) releaseTemplate(filename string, w io.Writer) error {
	_, err := os.Stat(filename)

	if err == nil {
		return output.ErrSkip
	}

	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if _, err = w.Write([]byte(output.Preamble("# "))); err != nil {
		return err
	}

	tmpl, err := template.New("config").Parse(releaseTemplateStr)
	if err != nil {
		return err
	}

	return tmpl.Execute(w, o.meta)
}

func (o *Output) config(w io.Writer) error {
	if _, err := w.Write([]byte(output.Preamble("# "))); err != nil {
		return err
	}

	tmpl, err := template.New("config").Parse(configStr)
	if err != nil {
		return err
	}

	return tmpl.Execute(w, o.meta)
}

func (o *Output) changelogTemplate(w io.Writer) error {
	if _, err := w.Write([]byte(output.Preamble("<!-- ", " -->"))); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%s\n", templateStr); err != nil {
		return err
	}

	return nil
}

// Compiler is implemented by project blocks which support Dockerfile generate.
type Compiler interface {
	CompileRelease(*Output) error
}

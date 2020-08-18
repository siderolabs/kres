// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package release implements output for releases.
package release

import (
	"fmt"
	"io"
	"os"

	"github.com/talos-systems/kres/internal/output"
)

const (
	release  = "./hack/release.sh"
	config   = "./hack/git-chglog/config.yaml"
	template = "./hack/git-chglog/CHANGELOG.tpl.md"
)

const releaseStr = `
set -e

function changelog {
  if [ "$#" -eq 1 ]; then
    git-chglog --output CHANGELOG.md -c ./hack/git-chglog/config.yaml --tag-filter-pattern "^${1}" "${1}.0-alpha.1.."
  elif [ "$#" -eq 0 ]; then
    git-chglog --output CHANGELOG.md -c ./hack/git-chglog/config.yaml
  else
    echo 1>&2 "Usage: $0 changelog [tag]"
    exit 1
  fi
}

function release-notes {
  git-chglog --output ${1} -c ./hack/git-chglog/config.yaml "${2}"
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
  commit:       Create the official release commit message.
  cherry-pick:  Cherry-pick a commit into a release branch.
  changelog:    Update the specified CHANGELOG.
EOF

  exit 1
fi`

const configStr = `style: github
template: CHANGELOG.tpl.md
info:
  title: CHANGELOG
  repository_url: https://github.com/talos-systems/talos
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
      - BREAKING CHANGE`

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
	return []string{release, config, template}
}

// GenerateFile implements output.FileWriter interface.
func (o *Output) GenerateFile(filename string, w io.Writer) error {
	switch filename {
	case release:
		return o.release(w)
	case config:
		return o.config(w)
	case template:
		return o.template(w)
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

func (o *Output) config(w io.Writer) error {
	if _, err := w.Write([]byte(output.Preamble("# "))); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%s\n", configStr); err != nil {
		return err
	}

	return nil
}

func (o *Output) template(w io.Writer) error {
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

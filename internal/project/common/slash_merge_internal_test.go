// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-github/v92/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/siderolabs/kres/internal/project/meta"
)

// fakeEnvironmentAPI is a minimal fake of the GitHub environments API for a single environment.
type fakeEnvironmentAPI struct {
	env      map[string]any // nil if the environment doesn't exist
	policies []map[string]any
	calls    []string
	nextID   int64
	mu       sync.Mutex
}

//nolint:gocyclo,cyclop
func (f *fakeEnvironmentAPI) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	const prefix = "/repos/org/repo/environments/" + slashMergeEnvironment

	path := strings.TrimPrefix(req.URL.Path, "/api/v3")
	f.calls = append(f.calls, req.Method+" "+strings.TrimPrefix(path, prefix))

	writeJSON := func(v any) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(v) //nolint:errcheck,errchkjson
	}

	switch {
	case req.Method == http.MethodGet && path == prefix:
		if f.env == nil {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		writeJSON(f.env)
	case req.Method == http.MethodPut && path == prefix:
		var body map[string]any

		json.NewDecoder(req.Body).Decode(&body) //nolint:errcheck

		f.env = map[string]any{
			"name":                     slashMergeEnvironment,
			"can_admins_bypass":        body["can_admins_bypass"],
			"deployment_branch_policy": body["deployment_branch_policy"],
		}

		writeJSON(f.env)
	case req.Method == http.MethodGet && path == prefix+"/deployment-branch-policies":
		writeJSON(map[string]any{"total_count": len(f.policies), "branch_policies": f.policies})
	case req.Method == http.MethodPost && path == prefix+"/deployment-branch-policies":
		var body map[string]any

		json.NewDecoder(req.Body).Decode(&body) //nolint:errcheck

		f.nextID++
		policy := map[string]any{"id": f.nextID, "name": body["name"], "type": body["type"]}
		f.policies = append(f.policies, policy)

		writeJSON(policy)
	case req.Method == http.MethodDelete && strings.HasPrefix(path, prefix+"/deployment-branch-policies/"):
		id := strings.TrimPrefix(path, prefix+"/deployment-branch-policies/")

		for i, p := range f.policies {
			if fmt.Sprint(p["id"]) == id {
				f.policies = append(f.policies[:i], f.policies[i+1:]...)

				break
			}
		}

		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func TestEnableSlashMergeEnvironment(t *testing.T) {
	t.Parallel()

	correctEnv := map[string]any{
		"name":                     slashMergeEnvironment,
		"can_admins_bypass":        false,
		"deployment_branch_policy": map[string]any{"protected_branches": false, "custom_branch_policies": true},
	}

	for _, test := range []struct { //nolint:govet
		name          string
		env           map[string]any
		policies      []map[string]any
		expectedCalls []string
	}{
		{
			name: "missing",
			expectedCalls: []string{
				"GET ",
				"PUT ",
				"GET /deployment-branch-policies",
				"POST /deployment-branch-policies",
			},
		},
		{
			name:     "up to date",
			env:      correctEnv,
			policies: []map[string]any{{"id": 1, "name": "main", "type": "branch"}},
			expectedCalls: []string{
				"GET ",
				"GET /deployment-branch-policies",
			},
		},
		{
			name: "tampered",
			env: map[string]any{
				"name":                     slashMergeEnvironment,
				"can_admins_bypass":        true,
				"deployment_branch_policy": nil,
			},
			policies: []map[string]any{
				{"id": 1, "name": "*", "type": "branch"},
				{"id": 2, "name": "main", "type": "tag"},
			},
			expectedCalls: []string{
				"GET ",
				"PUT ",
				"GET /deployment-branch-policies",
				"DELETE /deployment-branch-policies/1",
				"DELETE /deployment-branch-policies/2",
				"POST /deployment-branch-policies",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			api := &fakeEnvironmentAPI{env: test.env, policies: test.policies, nextID: 100}

			srv := httptest.NewServer(api)
			t.Cleanup(srv.Close)

			client, err := github.NewClient(github.WithURLs(new(srv.URL+"/"), new(srv.URL+"/")))
			require.NoError(t, err)

			r := NewRepository(&meta.Options{
				MainBranch:         "main",
				GitHubOrganization: "org",
				GitHubRepository:   "repo",
			})

			require.NoError(t, r.enableSlashMergeEnvironment(client))

			assert.Equal(t, test.expectedCalls, api.calls)

			// the end state is always the same: admins can't bypass, only main can deploy
			assert.Equal(t, false, api.env["can_admins_bypass"])
			assert.Equal(t, map[string]any{"protected_branches": false, "custom_branch_policies": true}, api.env["deployment_branch_policy"])
			require.Len(t, api.policies, 1)
			assert.Equal(t, "main", api.policies[0]["name"])
			assert.Equal(t, "branch", api.policies[0]["type"])
		})
	}
}

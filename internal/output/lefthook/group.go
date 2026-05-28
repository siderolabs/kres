// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lefthook

// Group is a container for nested Jobs with its own parallel/piped semantics.
// Used to express "run these in parallel, then those sequentially" without
// touching the hook-level execution model.
type Group struct { //nolint:govet
	// Parallel is a pointer so an explicit `parallel: false` can be emitted
	// (the bool zero value would otherwise be suppressed by omitempty).
	Parallel *bool  `yaml:"parallel,omitempty"`
	Piped    bool   `yaml:"piped,omitempty"`
	Jobs     []*Job `yaml:"jobs,omitempty"`
}

// WithParallel toggles parallel execution of this group's jobs.
func (g *Group) WithParallel(parallel bool) *Group {
	g.Parallel = &parallel

	return g
}

// WithPiped enables piped (sequential, fail-fast, stdout-chained) execution.
func (g *Group) WithPiped(piped bool) *Group {
	g.Piped = piped

	return g
}

// Job appends a new job to the group and returns it for further configuration.
func (g *Group) Job() *Job {
	j := &Job{}

	g.Jobs = append(g.Jobs, j)

	return j
}

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lefthook

// Config is a declarative description of lefthook.yml contributions, suitable
// for unmarshalling from a kres.yaml block.
//
// Hooks reuse the very same builder types (Hook, Command, Job, Group) that
// serialize lefthook.yml, so the config is authored in lefthook's native schema
// and there is a single definitive set of types for both serializing the output
// and deserializing the config.
type Config struct {
	Hooks   map[string]*Hook `yaml:"hooks"`
	Enabled bool             `yaml:"enabled"`
}

// Compile merges the configured hooks into the output. It is a no-op when the
// config is disabled.
func (c Config) Compile(output *Output) error {
	if !c.Enabled {
		return nil
	}

	for name, cfgHook := range c.Hooks {
		if cfgHook == nil {
			continue
		}

		hook := output.Hook(name)

		if cfgHook.Parallel != nil {
			hook.WithParallel(*cfgHook.Parallel)
		}

		if cfgHook.Piped {
			hook.WithPiped(true)
		}

		for cmdName, cmd := range cfgHook.Commands {
			if cmd == nil {
				continue
			}

			*hook.Command(cmdName) = *cmd
		}

		for _, job := range cfgHook.Jobs {
			if job == nil {
				continue
			}

			// A named job wrapping a group is merged into the hook's group of
			// the same name, so a config can extend the shared fix/lint groups
			// emitted by the standard blocks instead of forking a new one.
			if job.Group != nil && job.Name != "" {
				group := hook.Group(job.Name)

				if job.Group.Parallel != nil {
					group.WithParallel(*job.Group.Parallel)
				}

				if job.Group.Piped {
					group.WithPiped(true)
				}

				group.Jobs = append(group.Jobs, job.Group.Jobs...)

				continue
			}

			hook.Jobs = append(hook.Jobs, job)
		}
	}

	return nil
}

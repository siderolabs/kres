// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package codecov

const configTemplate = `codecov:
require_ci_to_pass: false

coverage:
status:
  project:
    default:
      target: %d%%
      threshold: 0.5%%
      base: auto
      if_ci_failed: success
  patch: off

comment: false
`

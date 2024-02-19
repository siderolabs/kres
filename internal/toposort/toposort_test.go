// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package toposort_test

import (
	"testing"

	"github.com/siderolabs/kres/internal/toposort"
)

type testNode int

func (n testNode) Before(m testNode) bool {
	type pair = [2]testNode

	switch (pair{n, m}) {
	case pair{5, 11}:
		return true
	case pair{7, 11}:
		return true
	case pair{7, 8}:
		return true
	case pair{3, 8}:
		return true
	case pair{3, 10}:
		return true
	case pair{11, 2}:
		return true
	case pair{11, 10}:
		return true
	case pair{11, 9}:
		return true
	case pair{8, 9}:
		return true
	default:
		return false
	}
}

func TestSort(t *testing.T) {
	for range 10000 {
		testnodes := []testNode{
			testNode(10),
			testNode(2),
			testNode(5),
			testNode(3),
			testNode(11),
			testNode(8),
			testNode(9),
			testNode(7),
		}

		sorted, bad := toposort.Stable(testnodes)

		if bad != nil {
			t.Errorf("returned bad: %v", bad)
			t.Fail()
		}

		expected := []testNode{5, 3, 7, 11, 8, 10, 2, 9}

		for j := 0; j < len(sorted) || j < len(expected); j++ {
			if sorted[j] != expected[j] {
				t.Logf("%v", sorted)

				t.Errorf("%v != %v", sorted[j], expected[j])
				t.FailNow()
			}
		}
	}
}

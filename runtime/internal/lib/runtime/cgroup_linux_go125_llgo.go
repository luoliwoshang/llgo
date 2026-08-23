//go:build linux && go1.25

package runtime

import _ "unsafe"

//go:linkname cgroup_throw internal/runtime/cgroup.throw
func cgroup_throw(s string) {
	throw(s)
}

package recoverpkg

// Store records a panic recovered by this directly deferred cross-package call.
func Store(dst *any) {
	*dst = recover()
}

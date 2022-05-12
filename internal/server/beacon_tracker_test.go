package server

func restoreHooks() func() {
	n := now
	return func() {
		now = n
	}
}

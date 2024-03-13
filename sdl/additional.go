package sdl

// tern replaces cond ? v1 : v2
func tern[T any](cond bool, v1, v2 T) T {
	if cond {
		return v1
	}
	return v2
}

// TODO panics with "not implemeted"
func TODO() {
	panic("not implemeted")
}

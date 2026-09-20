package crypto

import "runtime"

// Zeroize overwrites b with zeros. It is best effort: Go's GC may have
// already copied the data, but this still shortens the lifetime of secrets.
func Zeroize(b []byte) {
	for i := range b {
		b[i] = 0
	}
	runtime.KeepAlive(b)
}

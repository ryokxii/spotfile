//go:build !darwin && !windows

package trash

// Trash is unavailable here; nothing is deleted.
func (System) Trash(string) error {
	return ErrUnsupported
}

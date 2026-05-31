//go:build !windows

package datalimiter

func (OSDeps) IsAdmin() bool {
	return false
}

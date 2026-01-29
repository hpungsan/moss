//go:build !windows

package ops

import (
	"os"
	"syscall"

	"github.com/hpungsan/moss/internal/errors"
)

// openFileNoFollow opens a file for writing with O_NOFOLLOW to prevent symlink attacks.
// This closes the TOCTOU gap between ValidatePath and file open.
func openFileNoFollow(path string, flag int, perm os.FileMode) (*os.File, error) {
	fd, err := syscall.Open(path, flag|syscall.O_NOFOLLOW, uint32(perm))
	if err != nil {
		if err == syscall.ELOOP {
			return nil, errors.NewInvalidRequest("cannot write to symlink")
		}
		return nil, err
	}
	return os.NewFile(uintptr(fd), path), nil
}

// openFileNoFollowRead opens a file for reading with O_NOFOLLOW to prevent symlink attacks.
// This closes the TOCTOU gap between ValidatePath and file open.
func openFileNoFollowRead(path string) (*os.File, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		if err == syscall.ELOOP {
			return nil, errors.NewInvalidRequest("cannot read from symlink")
		}
		if err == syscall.ENOENT {
			return nil, errors.NewFileNotFound(path)
		}
		return nil, err
	}
	return os.NewFile(uintptr(fd), path), nil
}

//go:build !windows

package ops

import (
	stderrors "errors"
	"os"
	"syscall"

	"github.com/hpungsan/moss/internal/errors"
)

// openFileNoFollow opens a file for writing with O_NOFOLLOW to prevent symlink attacks
// on the final path component. O_CLOEXEC prevents FD leaks across exec.
//
// Note: O_NOFOLLOW only protects the final component. Directory components are validated
// by ValidatePath (which requires files to be directly in allowed directories, disallowing
// nested paths that could be subject to directory symlink swaps).
func openFileNoFollow(path string, flag int, perm os.FileMode) (*os.File, error) {
	fd, err := syscall.Open(path, flag|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, uint32(perm))
	if err != nil {
		if stderrors.Is(err, syscall.ELOOP) {
			return nil, errors.NewInvalidRequest("cannot write to symlink")
		}
		return nil, err
	}
	return os.NewFile(uintptr(fd), path), nil
}

// openFileNoFollowRead opens a file for reading with O_NOFOLLOW to prevent symlink attacks
// on the final path component. O_CLOEXEC prevents FD leaks across exec.
//
// Note: O_NOFOLLOW only protects the final component. Directory components are validated
// by ValidatePath (which requires files to be directly in allowed directories, disallowing
// nested paths that could be subject to directory symlink swaps).
func openFileNoFollowRead(path string) (*os.File, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		if stderrors.Is(err, syscall.ELOOP) {
			return nil, errors.NewInvalidRequest("cannot read from symlink")
		}
		if stderrors.Is(err, syscall.ENOENT) {
			return nil, errors.NewFileNotFound(path)
		}
		return nil, err
	}
	return os.NewFile(uintptr(fd), path), nil
}

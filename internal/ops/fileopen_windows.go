//go:build windows

package ops

import (
	"os"

	"github.com/hpungsan/moss/internal/errors"
)

// openFileNoFollow opens a file for writing.
// On Windows, O_NOFOLLOW is not available. Symlink attacks are less common
// on Windows due to privilege requirements for symlink creation.
// ValidatePath still checks for symlinks before we get here.
func openFileNoFollow(path string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(path, flag, perm)
}

// openFileNoFollowRead opens a file for reading.
// On Windows, O_NOFOLLOW is not available. See openFileNoFollow for details.
func openFileNoFollowRead(path string) (*os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NewFileNotFound(path)
		}
		return nil, err
	}
	return f, nil
}

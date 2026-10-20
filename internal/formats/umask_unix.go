//go:build unix

package formats

import (
	"os"
	"syscall"
)

func filePerm(targetPath string) os.FileMode {
	if info, err := os.Stat(targetPath); err == nil {
		return info.Mode().Perm()
	}
	mask := syscall.Umask(0)
	syscall.Umask(mask)
	return os.FileMode(0666 &^ mask)
}

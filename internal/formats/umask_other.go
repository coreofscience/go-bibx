//go:build !unix

package formats

import "os"

func filePerm(targetPath string) os.FileMode {
	if info, err := os.Stat(targetPath); err == nil {
		return info.Mode().Perm()
	}
	return 0666
}

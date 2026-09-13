//go:build !plan9

package memfs

import "syscall"

var errNotEmpty error = syscall.ENOTEMPTY

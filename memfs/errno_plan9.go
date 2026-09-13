package memfs

import "syscall"

// plan9 has no ENOTEMPTY.
var errNotEmpty error = syscall.NewError("directory not empty")

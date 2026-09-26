//go:build !unix

package app

import "os"

// no file locking here; the list is still replaced atomically, so a lost race drops an entry, never corrupts the file
func lockFile(file *os.File, exclusive bool) error { return nil }

func unlockFile(file *os.File) error { return nil }

//go:build linux

package core

import (
	"os"
	"syscall"
)

// SendFileLinux uses the sendfile system call for true zero-copy on Linux
func SendFileLinux(dst *os.File, src *os.File, offset, count int64) (int64, error) {
	var written int64
	remaining := count
	srcFd := int(src.Fd())
	dstFd := int(dst.Fd())

	for remaining > 0 {
		// Sendfile can transfer at most 0x7ffff000 bytes at once
		const maxSendfile = 0x7ffff000
		toSend := remaining
		if toSend > maxSendfile {
			toSend = maxSendfile
		}

		n, err := syscall.Sendfile(dstFd, srcFd, nil, int(toSend))
		if n > 0 {
			written += int64(n)
			remaining -= int64(n)
		}
		if err != nil {
			if err == syscall.EAGAIN {
				continue // Retry on EAGAIN
			}
			return written, err
		}
		if n == 0 {
			break // EOF
		}
	}

	return written, nil
}

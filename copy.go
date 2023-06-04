package main

import (
	"io"
)

func Copy(dst io.Writer, src io.Reader) error {
	buf := make([]byte, 0, 1)
	_, err := io.CopyBuffer(dst, src, buf[:cap(buf)])
	return err
}

package main

import (
	"io"
)

func Copy(reader io.Reader, writer io.Writer) error {
	buf := BufferPool.Get()
	defer BufferPool.Put(buf)

	_, err := io.CopyBuffer(writer, reader, buf[:cap(buf)])
	return err
}

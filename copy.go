package main

import (
	"io"
)

func Copy(reader io.Reader, writer io.Writer) error {
	b := make([]byte, 1024)

	for {
		n, err := reader.Read(b)
		if n > 0 {
			res := make([]byte, n)
			// Copy the buffer that it doesn't get changed while read by the recipient.
			copy(res, b[:n])
			writer.Write(res)
		}
		if err != nil {
			if err == io.EOF || err == io.ErrClosedPipe {
				return nil
			}
			return err
		}
	}
}

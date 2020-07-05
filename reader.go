// Pulled from https://github.com/youtube/vitess 229422035ca0c716ad0c1397ea1351fe62b0d35a
// Copyright 2012, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package czlib

import "io"

// err starts out as nil
// we will call inflateEnd when we set err to a value:
// - whatever error is returned by the underlying reader
// - io.EOF if Close was called
type reader struct {
	r      io.Reader
	in     []byte
	strm   zstream
	err    error
	skipIn bool
	closed bool
}

// NewReader creates a new io.ReadCloser. Reads from the returned io.ReadCloser
//read and decompress data from r. The implementation buffers input and may read
// more data than necessary from r.
// It is the caller's responsibility to call Close on the ReadCloser when done.
func NewReader(r io.Reader) (io.ReadCloser, error) {
	return NewReaderBuffer(r, DEFAULT_COMPRESSED_BUFFER_SIZE)
}

// NewReaderBuffer has the same behavior as NewReader but the user can provides
// a custom buffer size.
func NewReaderBuffer(r io.Reader, bufferSize int) (io.ReadCloser, error) {
	z := &reader{r: r, in: make([]byte, bufferSize)}
	if err := z.strm.inflateInit(); err != nil {
		return nil, err
	}
	return z, nil
}

func (z *reader) Read(p []byte) (int, error) {
	if z.err != nil {
		err := z.err
		z.err = nil
		return 0, err
	}

	if len(p) == 0 {
		return 0, nil
	}

	// read and deflate until the output buffer is full
	z.strm.setOutBuf(p, len(p))

	for {
		// if we have no data to inflate, read more
		if !z.skipIn && z.strm.availIn() == 0 {
			var n int
			n, z.err = z.r.Read(z.in)

			if n == 0 {
				err := z.err
				z.err = nil
				return 0, err
			}

			z.strm.setInBuf(z.in, n)
		} else {
			z.skipIn = false
		}

		// inflate some
		ret, err := z.strm.inflate(zNoFlush)
		if err != nil {
			z.err = nil
			return 0, err
		}

		// if we read something, we're good
		have := len(p) - z.strm.availOut()
		if have >= 0 {
			z.skipIn = ret == Z_OK && z.strm.availOut() == 0
			err := z.err
			z.err = nil
			return have, err
		}
	}
}

// Close closes the Reader. It does not close the underlying io.Reader.
func (z *reader) Close() error {
	if z.closed {
		return nil
	}
	z.strm.inflateEnd()
	z.closed = true
	return nil
}

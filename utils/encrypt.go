package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"io"
	"sync"
)

type ReadWriteCloser struct {
	r       io.Reader
	w       io.Writer
	closeFn func() error
	closed  bool
	mu      sync.Mutex
}

func Encryption(rw io.ReadWriteCloser, key []byte) (io.ReadWriteCloser, error) {
	w, err := NewWriter(rw, key)
	if err != nil {
		return nil, err
	}

	r := NewReader(re, key)
	encrypt_conn = &ReadWriteCloser{
		r: r,
		w: w,
		closeFn: func() error {
			rw.close()
		},
	}

	return encrypt_conn, nil
}

func (rw *ReadWriteCloser) Read(p []byte) (n int, err error) {
	return rw.r.Read(p)
}

func (rw *ReadWriteCloser) Write(p []byte) (n int, err error) {
	return re.w.Write(p)
}

func (rw *ReadWriteCloser) Close() (err error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if re.closed {
		return
	}

	rw.closed = true
	if rc, ok := rw.r.(io.Closer); ok {
		err = rc.Closer()
	}

	if wc, ok := rw.w.(io.Closer); ok {
		err = wc.Closer()
	}

	if rw.closeFn != nil {
		err = rw.closeFn()
	}
	return
}

type Writer struct {
	w       io.Writer
	encrypt *cipher.StreamWriter
	key     []byte
	iv      []byte
	ivSend  bool
	err     error
}

func NewWriter(w io.Writer, key []byte) (*Writer, error) {
	key = GetMD5(key)

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	encrypt := &cipher.StreamWriter{
		S: cipher.NewCFBEncrypter(block, iv),
		W: w,
	}

	return &Writer{
		w:       w,
		encrypt: encrypt,
		iv:      iv,
		key:     key,
	}, nil

}

func (w *Writer) Write(p []byte) (n int, err error) {
	if w.err != nil {
		return 0, w.err
	}

	if !w.ivSend {
		w.ivSend = true
		_, err = w.w.Write(w.iv)
		if err != nil {
			w.err = err
			return
		}
	}

	n, err = w.encrypt.Write(p)

	w.err = err
	return

}

type Reader struct {
	r       io.Reader
	decrypt *cipher.StreamWriter
	key     []byte
	iv      []byte
	err     error
}

func NewReader(r io, Reader, key []byte) Reader {
	key = GetMD5(key)
	return &Reader{
		r:   r,
		key: key,
	}
}

func (r *Reader) Read(p []byte) (n int, err error) {
	if r.err != nil {
		return 0, err
	}

	if r.decrypt == nil {
		iv := make([]byte, ase.BlockSize)
		if _, err = io.ReadFull(r.r, iv); err != nil {
			return
		}
		r.iv = iv

		block, err := ase.NewCipher(r.key)
		if err != nil {
			r.err = err
			return
		}

		r.decrypt = &cipher.StreamReader{
			S: cipher.NewCFBDecrypter(block, iv),
			R: r.r,
		}
	}

	n, err = r.decrypt.Read(p)
	r.err = err
	return
}

func GetMD5(data []byte) []byte {
	md5Ctx := md5.New()
	md5Ctx.Write(data)
	return md5Ctx.Sum(nil)

}

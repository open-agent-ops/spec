package conformance

import (
	"errors"
	"io"
	"strings"
	"testing"
)

type closingReader struct {
	io.Reader
	closed bool
}

type failingReadCloser struct {
	closed     bool
	closeError bool
}

func (r *failingReadCloser) Read([]byte) (int, error) {
	return 0, errors.New("private diagnostic must not escape")
}
func (r *failingReadCloser) Close() error {
	r.closed = true
	if r.closeError {
		return errors.New("private close diagnostic")
	}
	return nil
}
func TestReaderFailureCleanup(t *testing.T) {
	for _, failClose := range []bool{false, true} {
		r := &failingReadCloser{closeError: failClose}
		data, err := Read(r)
		if !r.closed || data != nil || err != ErrInput {
			t.Fatal("unsafe read failure")
		}
	}
	if _, err := Read(nil); err != ErrInput {
		t.Fatal("nil reader")
	}
}

func (r *closingReader) Close() error { r.closed = true; return nil }

func TestInputGuards(t *testing.T) {
	for _, s := range []string{`{"a":1,"a":2}`, `{} {}`, `{"a":`, strings.Repeat(" ", MaxBytes+1)} {
		if _, err := Decode([]byte(s)); err == nil {
			t.Fatal("invalid input accepted")
		}
	}
	tests := []struct {
		p        profile
		at, over string
	}{
		{profile{2, 99, 99, 99}, `[0]`, `[[0]]`},
		{profile{99, 3, 99, 99}, `[0,0]`, `[0,0,0]`},
		{profile{99, 99, 2, 99}, `[0,0]`, `[0,0,0]`},
		{profile{99, 99, 99, 2}, `"é"`, `"éx"`},
	}
	for _, tc := range tests {
		if _, err := decode([]byte(tc.at), tc.p); err != nil {
			t.Fatal("at limit rejected")
		}
		if _, err := decode([]byte(tc.over), tc.p); err == nil {
			t.Fatal("over limit accepted")
		}
	}
	if !ValidPath(strings.Repeat("a", 256)) || ValidPath(strings.Repeat("a", 257)) {
		t.Fatal("path bytes")
	}
	if !ValidPath(strings.Repeat("a/", 15)+"a") || ValidPath(strings.Repeat("a/", 16)+"a") {
		t.Fatal("path depth")
	}
	for _, s := range []string{"", "/a", "a/../b", "a/./b", "a\\b", "a//b"} {
		if ValidPath(s) {
			t.Fatal("hostile path")
		}
	}
	for _, n := range []int{MaxBytes, MaxBytes + 1} {
		r := &closingReader{Reader: strings.NewReader(strings.Repeat("x", n))}
		_, err := Read(r)
		if !r.closed || (err == nil) != (n == MaxBytes) {
			t.Fatal("reader boundary/cleanup")
		}
	}
}

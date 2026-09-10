//go:build !unix

package releaseops

import "os/exec"

// Native publisher/mirror jobs use Unix managed runners. Other platforms must
// not start an unbounded Git child tree.
func configureGitCancellation(*exec.Cmd) error { return ErrAuthority }

// Package cli wires the kp commands to the terminal.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/azcpcf/kp-go/internal/kill"
	"github.com/azcpcf/kp-go/internal/scan"
)

// version is set at build time with -ldflags -X.
var version = "dev"

// options holds the dependencies of the command tree.
type options struct {
	scan    func(proto string) ([]scan.Listener, error)
	killer  *kill.Killer
	version string
}

// exitError carries a process exit code. A nil err means the failure was
// already reported to stderr.
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string {
	if e.err != nil {
		return e.err.Error()
	}
	return fmt.Sprintf("exit status %d", e.code)
}

func (e *exitError) Unwrap() error {
	return e.err
}

// Execute runs kp and exits with its status code.
func Execute() {
	os.Exit(Main(os.Args[1:], os.Stdout, os.Stderr))
}

// Main runs kp with args and returns a process exit code: 0 on success, 1 on
// runtime errors and 2 on usage errors.
func Main(args []string, out, errw io.Writer) int {
	return run(args, out, errw, options{
		scan:    scan.List,
		killer:  kill.New(),
		version: version,
	})
}

// run builds and executes the command tree. Runtime failures must come back as
// *exitError; every other error is treated as a usage error.
func run(args []string, out, errw io.Writer, opts options) int {
	root := newRoot(opts)
	root.SetArgs(args)
	root.SetOut(out)
	root.SetErr(errw)

	_, err := root.ExecuteC()
	if err == nil {
		return 0
	}

	var ee *exitError
	if errors.As(err, &ee) {
		if ee.err != nil {
			fmt.Fprintf(errw, "error: %v\n", ee.err)
		}
		return ee.code
	}

	fmt.Fprintf(errw, "error: %v\n", err)
	fmt.Fprintln(errw, "run 'kp --help' for usage")
	return 2
}

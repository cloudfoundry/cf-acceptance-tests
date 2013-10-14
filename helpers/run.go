package helpers

import (
	"io"
	"os"
	"os/exec"

	"github.com/vito/cmdtest"
)

func Run(executable string, args ...string) *cmdtest.Session {
	cmd := exec.Command(executable, args...)

	sess, err := cmdtest.StartWrapped(cmd, teeStdout, teeStderr)
	if err != nil {
		panic(err)
	}

	return sess
}

func Curl(args ...string) *cmdtest.Session {
	args = append([]string{"-s"}, args...)
	return Run("curl", args...)
}

func Cf(args ...string) *cmdtest.Session {
	return Run("go-cf", args...)
}

func teeStdout(out io.Reader) io.Reader {
	if verboseOutputEnabled() {
		return io.TeeReader(out, os.Stdout)
	} else {
		return out
	}
}

func teeStderr(out io.Reader) io.Reader {
	if verboseOutputEnabled() {
		return io.TeeReader(out, os.Stderr)
	} else {
		return out
	}
}

func verboseOutputEnabled() bool {
	verbose := os.Getenv("CF_VERBOSE_OUTPUT")
	return verbose == "yes" || verbose == "true"
}

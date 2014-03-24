package runner

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/vito/cmdtest"
)

var SessionStarter = func(cmd *exec.Cmd) (*cmdtest.Session, error) {
	return cmdtest.StartWrapped(cmd, teeStdout, teeStderr)
}

func Run(executable string, args ...string) *cmdtest.Session {
	cmd := exec.Command(executable, args...)

	if verboseOutputEnabled() {
		fmt.Println("\n", "> ", strings.Join(cmd.Args, " "))
	}

	sess, err := SessionStarter(cmd)
	if err != nil {
		panic(err)
	}

	return sess
}

func Curl(args ...string) *cmdtest.Session {
	args = append([]string{"-s"}, args...)
	return Run("curl", args...)
}

func teeStdout(out io.Writer) io.Writer {
	if verboseOutputEnabled() {
		return io.MultiWriter(out, os.Stdout)
	} else {
		return out
	}
}

func teeStderr(out io.Writer) io.Writer {
	if verboseOutputEnabled() {
		return io.MultiWriter(out, os.Stderr)
	} else {
		return out
	}
}

func verboseOutputEnabled() bool {
	verbose := os.Getenv("CF_VERBOSE_OUTPUT")
	return verbose == "yes" || verbose == "true"
}

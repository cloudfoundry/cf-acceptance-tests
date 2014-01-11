package helpers

import (
	"io"
	"os"
	"os/exec"
	"strconv"

	"github.com/onsi/ginkgo/config"
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
	trace_file := os.Getenv("CF_TRACE_BASENAME")
	if trace_file != "" {
		os.Setenv("CF_TRACE", trace_file+strconv.Itoa(parallelNode())+".txt")
	}

	return Run("gcf", args...)
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

func parallelNode() int {
	return config.GinkgoConfig.ParallelNode
}

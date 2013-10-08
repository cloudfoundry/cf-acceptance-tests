package helpers

import (
	"github.com/vito/cmdtest"
)

func Run(executable string, args ...string) *cmdtest.Session {
	sess, err := cmdtest.Start(executable, args...)
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

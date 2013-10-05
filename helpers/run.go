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

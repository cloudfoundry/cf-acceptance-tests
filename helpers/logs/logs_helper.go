package logs

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

func Tail(useLogCache bool, appName string) func() *gbytes.Buffer {
	return func() *gbytes.Buffer {
		var session *Session
		if useLogCache {
			session = cf.Cf("tail", "--envelope-class=logs", appName, "--lines", "125")
		} else {
			session = cf.Cf("logs", "--recent", appName)
		}

		return session.Wait().Out
	}
}

func TailFollow(useLogCache bool, appName string) *Session {
	if useLogCache {
		return cf.Cf("tail", "--envelope-class=logs", "--follow", appName)
	}

	return cf.Cf("logs", appName)
}

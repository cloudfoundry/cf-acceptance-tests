package internal

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/internal"
	"github.com/onsi/gomega/gexec"
	"time"
	. "github.com/onsi/gomega"
)

func CfAuth(cmdStarter internal.Starter, reporter internal.Reporter, user string, password string, ) *gexec.Session {
	var auth *gexec.Session
	var err error

	retries := 2
	for i := 1; i <= retries; i++ {
		auth, err = cmdStarter.Start(reporter, "cf", "auth", user, password)
		if err != nil {
			panic(err)
		}

		if i < retries {
			// retry timeouts if not final retry
			failures := InterceptGomegaFailures(func() {
				auth.Wait(5)
			})
			if len(failures) != 0 {
				continue
			}
		} else {
			auth.Wait(5)
		}

		returnVal := auth.ExitCode()
		if returnVal == 0 {
			return auth
		}
		time.Sleep(1 * time.Second)
	}

	return auth
}

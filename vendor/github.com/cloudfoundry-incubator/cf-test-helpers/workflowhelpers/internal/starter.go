package internal

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/commandstarter"
	"github.com/onsi/gomega/gexec"
)

type starter interface {
	Start(commandstarter.Reporter, string, ...string) (*gexec.Session, error)
}

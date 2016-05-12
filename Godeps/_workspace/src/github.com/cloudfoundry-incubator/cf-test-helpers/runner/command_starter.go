package runner

import (
	"os/exec"
	"time"

	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

type CommandStarter struct {
	reporter Reporter
}

func NewCommandStarter() *CommandStarter {
	return &CommandStarter{
		reporter: NewDefaultReporter(),
	}
}

func NewCommandStarterWithReporter(reporter Reporter) *CommandStarter {
	return &CommandStarter{
		reporter: reporter,
	}
}

func (r *CommandStarter) Start(executable string, args ...string) *gexec.Session {
	cmd := exec.Command(executable, args...)
	r.reporter.Report(time.Now(), cmd)

	sess, err := gexec.Start(CommandInterceptor(cmd), ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	return sess
}

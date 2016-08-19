package runner

import (
	"os/exec"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega/gexec"
)

var CommandInterceptor = func(cmd *exec.Cmd) *exec.Cmd {
	return cmd
}

type CommandStarter struct {
}

func NewCommandStarter() *CommandStarter {
	return &CommandStarter{}
}

func (r *CommandStarter) Start(reporter Reporter, executable string, args ...string) (*gexec.Session, error) {
	cmd := exec.Command(executable, args...)
	reporter.Report(time.Now(), cmd)

	sess, err := gexec.Start(CommandInterceptor(cmd), ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)

	return sess, err
}

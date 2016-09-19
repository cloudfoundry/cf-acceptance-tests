package fakes

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/internal"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega/gexec"
)

type callToStartMethod struct {
	Executable string
	Args       []string
	Reporter   internal.Reporter
}

type startMethodStub struct {
	Output    string
	Err       error
	ExitCode  int
	SleepTime int
}

type FakeCmdStarter struct {
	CalledWith        []callToStartMethod
	ToReturn          []startMethodStub
	TotalCallsToStart int
}

func NewFakeCmdStarter() *FakeCmdStarter {
	return &FakeCmdStarter{
		CalledWith: []callToStartMethod{},
		ToReturn:   make([]startMethodStub, 10),
	}
}

func (s *FakeCmdStarter) Start(reporter internal.Reporter, executable string, args ...string) (*gexec.Session, error) {
	output := s.ToReturn[s.TotalCallsToStart].Output
	if output == "" {
		output = `\{\}`
	}
	sleepTime := s.ToReturn[s.TotalCallsToStart].SleepTime
	exitCode := s.ToReturn[s.TotalCallsToStart].ExitCode
	err := s.ToReturn[s.TotalCallsToStart].Err

	s.TotalCallsToStart += 1

	callToStart := callToStartMethod{
		Executable: executable,
		Args:       args,
		Reporter:   reporter,
	}
	s.CalledWith = append(s.CalledWith, callToStart)

	reporter.Report(time.Now(), exec.Command(executable, args...))
	cmd := exec.Command(
		"bash",
		"-c",
		fmt.Sprintf(
			"echo %s; sleep %d; exit %d",
			output,
			sleepTime,
			exitCode,
		),
	)
	session, _ := gexec.Start(cmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	return session, err
}

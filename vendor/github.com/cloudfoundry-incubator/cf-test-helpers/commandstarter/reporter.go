package runner

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
)

const timeFormat = "2006-01-02 15:04:05.00 (MST)"

type Reporter interface {
	Report(time.Time, *exec.Cmd)
}

type DefaultReporter struct{}

func NewDefaultReporter() *DefaultReporter {
	return &DefaultReporter{}
}

func (r *DefaultReporter) Report(startTime time.Time, cmd *exec.Cmd) {
	startColor := ""
	endColor := ""
	if !config.DefaultReporterConfig.NoColor {
		startColor = "\x1b[32m"
		endColor = "\x1b[0m"
	}
	fmt.Fprintf(ginkgo.GinkgoWriter, "\n%s[%s]> %s %s\n", startColor, startTime.UTC().Format(timeFormat), strings.Join(cmd.Args, " "), endColor)
}

package cli_version_check

import (
	"regexp"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/cli_version_check/cli_version"
)

type CliVersionCheck struct {
	cliVersion cli_version.CliVersion
}

func NewCliVersionCheck(c cli_version.CliVersion) CliVersionCheck {
	return CliVersionCheck{
		cliVersion: c,
	}
}

func (c CliVersionCheck) AtLeast(min_version string) bool {
	return min_version <= c.GetCliVersion()
}

func (c CliVersionCheck) GetCliVersion() string {
	output := c.cliVersion.GetVersion()

	re := regexp.MustCompile(`[0-9]+\.[0-9]+\.[0-9]+(?:\.[0-9]+)?`)
	result := re.FindStringSubmatch(output)

	if len(result) == 0 {
		return output
	}

	return result[0]
}

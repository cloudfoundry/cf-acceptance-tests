package cli_version

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

type InstalledCliVersion interface {
	GetVersion() string
	SetFullVersionString(string)
}

type cliVersion struct {
	fullVersionStr string
}

func NewInstalledCliVersion() InstalledCliVersion {
	output, err := exec.Command("cf", "-v").Output()
	if err != nil {
		fmt.Println("Error trying to determine CF CLI version:", err)
		os.Exit(1)
	}

	return &cliVersion{string(output)}
}

func (c *cliVersion) GetVersion() string {
	re := regexp.MustCompile(`[0-9]+\.[0-9]+\.[0-9]+(?:\.[0-9]+)?`)
	result := re.FindStringSubmatch(c.fullVersionStr)

	if len(result) == 0 {
		return c.fullVersionStr
	}

	return result[0]
}

func (c *cliVersion) SetFullVersionString(ver string) {
	c.fullVersionStr = ver
}

package cli_version_check

import (
	"errors"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type CliVersionCheck struct {
	BuildFromSource bool
	Revisions       []int
}

func GetInstalledCliVersionString() (string, error) {
	rawVersion, err := exec.Command("cf", "-v").Output()
	if err != nil {
		return "", errors.New("Error trying to determine CF CLI version:" + err.Error())
	}

	return string(rawVersion), nil
}

func ParseRawCliVersionString(rawVersion string) CliVersionCheck {
	if strings.Contains(rawVersion, "BUILT_FROM_SOURCE") {
		return CliVersionCheck{Revisions: []int{}, BuildFromSource: true}
	}

	re := regexp.MustCompile(`[0-9]+\.[0-9]+\.[0-9]+(?:\.[0-9]+)?`)
	result := re.FindStringSubmatch(rawVersion)

	if len(result) == 0 {
		return CliVersionCheck{Revisions: []int{}}
	}

	return CliVersionCheck{Revisions: parseRevisions(result[0])}
}

func (c CliVersionCheck) AtLeast(min_version CliVersionCheck) bool {
	if c.BuildFromSource {
		return true
	}

	if len(c.Revisions) > 0 && len(min_version.Revisions) == 0 {
		return true
	} else if len(min_version.Revisions) > 0 && len(c.Revisions) == 0 {
		for _, v := range min_version.Revisions {
			if v != 0 {
				return false
			}
		}
		return true
	}

	if c.Revisions[0] < min_version.Revisions[0] {
		return false
	} else if (len(c.Revisions) > 1 || len(min_version.Revisions) > 1) && c.Revisions[0] == min_version.Revisions[0] {
		return CliVersionCheck{false, c.Revisions[1:]}.AtLeast(CliVersionCheck{false, min_version.Revisions[1:]})
	}

	return true
}

func parseRevisions(extractedVersionStr string) []int {
	var revisions []int
	split := strings.Split(extractedVersionStr, ".")

	for _, v := range split {
		parsed, _ := strconv.Atoi(v)
		revisions = append(revisions, parsed)
	}

	return revisions
}

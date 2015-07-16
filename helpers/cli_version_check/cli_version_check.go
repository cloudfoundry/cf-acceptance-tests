package cli_version_check

import (
	"strconv"
	"strings"
)

type CliVersionCheck struct {
	buildFromSource bool
	revisions       []int
}

func NewCliVersionCheck(versionStr string) CliVersionCheck {
	c := CliVersionCheck{
		buildFromSource: false,
		revisions:       []int{},
	}

	if strings.Contains(versionStr, "BUILT_FROM_SOURCE") {
		c.buildFromSource = true
		return c
	}

	split := strings.Split(versionStr, ".")

	for _, v := range split {
		parsed, _ := strconv.Atoi(v)
		c.revisions = append(c.revisions, parsed)
	}

	return c
}

func (c CliVersionCheck) AtLeast(min_version CliVersionCheck) bool {
	if c.buildFromSource {
		return true
	}

	if len(c.revisions) > 0 && len(min_version.revisions) == 0 {
		return true
	} else if len(min_version.revisions) > 0 && len(c.revisions) == 0 {
		for _, v := range min_version.revisions {
			if v != 0 {
				return false
			}
		}
		return true
	}

	if c.revisions[0] < min_version.revisions[0] {
		return false
	} else if (len(c.revisions) > 1 || len(min_version.revisions) > 1) && c.revisions[0] == min_version.revisions[0] {
		return CliVersionCheck{false, c.revisions[1:]}.AtLeast(CliVersionCheck{false, min_version.revisions[1:]})
	}

	return true
}

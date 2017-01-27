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
	rawVersion, err := exec.Command("cf", "-v").CombinedOutput()
	if err != nil {
		return "", errors.New("Error trying to determine CF CLI version:" + err.Error() + "; cf -v Output:" + string(rawVersion))
	}

	return string(rawVersion), nil
}

func ParseRawCliVersionString(rawVersion string) CliVersionCheck {
	if strings.Contains(rawVersion, "BUILT_FROM_SOURCE") {
		return CliVersionCheck{Revisions: []int{}, BuildFromSource: true}
	}

	re := regexp.MustCompile(`[0-9]+(\.[0-9]+)*`)
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

	c.Revisions, min_version.Revisions = zeroPadToLongerArray(c.Revisions, min_version.Revisions)

	for i := 0; i < len(c.Revisions); i++ {
		if c.Revisions[i] == min_version.Revisions[i] {
			continue
		}

		return c.Revisions[i] > min_version.Revisions[i]
	}

	return true
}

func zeroPadToLongerArray(a1, a2 []int) (paddedA1, paddedA2 []int) {
	if len(a1) > len(a2) {
		paddedShorter := zeroPadToLength(a2, len(a1))
		return a1, paddedShorter
	} else {
		paddedShorter := zeroPadToLength(a1, len(a2))
		return paddedShorter, a2
	}
}

func zeroPadToLength(array []int, newLength int) []int {
	paddedArray := make([]int, newLength)
	for i, val := range array {
		paddedArray[i] = val
	}
	return paddedArray
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

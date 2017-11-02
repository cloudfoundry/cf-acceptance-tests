package download

import (
	"fmt"
	"regexp"

	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/config"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
)

func WithRedirect(url, path string, config config.CatsConfig) error {
	oauthToken := v3_helpers.GetAuthToken()
	downloadCurl := helpers.Curl(
		config,
		"-v", fmt.Sprintf("%s%s", config.GetApiEndpoint(), url),
		"-H", fmt.Sprintf("Authorization: %s", oauthToken),
		"-f",
	).Wait(config.DefaultTimeoutDuration())
	if downloadCurl.ExitCode() != 0 {
		return fmt.Errorf("curl exited with code: %d", downloadCurl.ExitCode())
	}

	locationHeaderRegex, err := regexp.Compile("Location: (.*)\r\n")
	if err != nil {
		return err
	}

	matches := locationHeaderRegex.FindStringSubmatch(string(downloadCurl.Err.Contents()))
	if len(matches) < 2 {
		return fmt.Errorf("didn't find location for redirect")
	}

	redirectURI := matches[1]
	downloadCurl = helpers.Curl(
		config,
		"-v", redirectURI,
		"--output", path,
		"-f",
	).Wait(config.DefaultTimeoutDuration())
	if downloadCurl.ExitCode() != 0 {
		return fmt.Errorf("curl exited with code: %d", downloadCurl.ExitCode())
	}
	return nil
}

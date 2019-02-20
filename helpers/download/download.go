package download

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"

	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/config"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
)

const maxNumRedirects = 10 // This is the same default Go uses with in its http library: https://godoc.org/net/http#Get

func WithRedirect(url, path string, config config.CatsConfig) error {
	oauthToken := v3_helpers.GetAuthToken()
	downloadCurl := helpers.Curl(
		config,
		"-v", fmt.Sprintf("%s%s%s", config.Protocol(), config.GetApiEndpoint(), url),
		"-H", fmt.Sprintf("Authorization: %s", oauthToken),
		"-f",
	).Wait()
	if downloadCurl.ExitCode() != 0 {
		return fmt.Errorf("curl exited with code: %d", downloadCurl.ExitCode())
	}

	isRedirect, redirectURI, err := CheckRedirect(string(downloadCurl.Err.Contents()))
	if err != nil {
		return err
	}
	if !isRedirect {
		ioutil.WriteFile(path, downloadCurl.Out.Contents(), 0644)
		return nil
	}
	for i := 0; i < maxNumRedirects; i++ {
		downloadCurl := helpers.Curl(
			config,
			"-v", redirectURI,
			"-f",
		).Wait()
		if downloadCurl.ExitCode() != 0 {
			return fmt.Errorf("curl exited with code: %d", downloadCurl.ExitCode())
		}

		isRedirect, redirectURI, err = CheckRedirect(string(downloadCurl.Err.Contents()))
		if err != nil {
			return err
		}
		if !isRedirect {
			ioutil.WriteFile(path, downloadCurl.Out.Contents(), 0644)
			return nil
		}
	}
	return fmt.Errorf("Only %v redirects allowed", maxNumRedirects)
}

func CheckRedirect(curlOutput string) (bool, string, error) {
	statusCodePattern := `HTTP/\d(?:\.\d)? (\d{3})[A-Za-z \-]*`
	statusCodeMatches := regexp.MustCompile(statusCodePattern).FindStringSubmatch(curlOutput)
	if len(statusCodeMatches) != 2 {
		return false, "", fmt.Errorf("Unexpected output from curl. Was expecting %v in the following output: %v", statusCodePattern, curlOutput)
	}
	statusCode, e := strconv.Atoi(statusCodeMatches[1])
	if e != nil {
		return false, "", fmt.Errorf("Unexpected status code from curl: %v", e.Error())
	}
	if statusCode > 300 && statusCode < 400 {
		matches := regexp.MustCompile("(?i)Location: (.*)\r\n").FindStringSubmatch(curlOutput)
		if len(matches) != 2 {
			return false, "", fmt.Errorf("Got redirect status code %v, but found no Location in curl output  %v", statusCode, len(matches))
		}
		return true, matches[1], nil
	}
	return false, "", nil
}

package app_helpers

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/config"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

type AppDroplet struct {
	appGuid string
	config  config.CatsConfig
}

func NewAppDroplet(appGuid string, config config.CatsConfig) *AppDroplet {
	return &AppDroplet{
		appGuid: appGuid,
		config:  config,
	}
}

func (droplet *AppDroplet) DownloadTo(downloadPath string) (dropletTarballPath string) {
	dropletTarballPath = fmt.Sprintf("%s.tar.gz", downloadPath)
	downloadUrl := fmt.Sprintf("/v2/apps/%s/droplet/download", droplet.appGuid)

	redirectFollower := NewRedirectFollower(droplet.config)
	redirectFollower.FollowRedirectWithoutHeaders(downloadUrl, dropletTarballPath)
	return
}

func (droplet *AppDroplet) UploadFrom(uploadPath string) {
	token := v3_helpers.GetAuthToken()
	uploadURL := fmt.Sprintf("%s%s/v2/apps/%s/droplet/upload", droplet.config.Protocol(), droplet.config.GetApiEndpoint(), droplet.appGuid)
	bits := fmt.Sprintf(`droplet=@%s`, uploadPath)
	curl := helpers.Curl(droplet.config, "-v", uploadURL, "-X", "PUT", "-F", bits, "-H", fmt.Sprintf("Authorization: %s", token)).Wait(droplet.config.DefaultTimeoutDuration())
	Expect(curl).To(Exit(0))

	var job struct {
		Metadata struct {
			Url string `json:"url"`
		} `json:"metadata"`
	}
	bytes := curl.Out.Contents()
	json.Unmarshal(bytes, &job)
	pollingUrl := job.Metadata.Url

	Eventually(func() *Session {
		return cf.Cf("curl", pollingUrl).Wait(droplet.config.DefaultTimeoutDuration())
	}, droplet.config.DefaultTimeoutDuration()).Should(Say("finished"))
}

type RedirectFollower struct {
	config config.CatsConfig
}

func NewRedirectFollower(config config.CatsConfig) *RedirectFollower {
	return &RedirectFollower{
		config: config,
	}
}

func (redirect *RedirectFollower) FollowRedirectWithoutHeaders(downloadURL, appDropletPathToCompressedFile string) {
	oauthToken := v3_helpers.GetAuthToken()
	downloadCurl := helpers.Curl(
		redirect.config,
		"-v", fmt.Sprintf("%s%s", redirect.config.GetApiEndpoint(), downloadURL),
		"-H", fmt.Sprintf("Authorization: %s", oauthToken),
		"-f",
	).Wait(redirect.config.DefaultTimeoutDuration())
	Expect(downloadCurl).To(Exit(0))

	curlOutput := string(downloadCurl.Err.Contents())
	locationHeaderRegex := regexp.MustCompile("Location: (.*)\r\n")
	redirectURI := locationHeaderRegex.FindStringSubmatch(curlOutput)[1]

	downloadCurl = helpers.Curl(
		redirect.config,
		"-v", redirectURI,
		"--output", appDropletPathToCompressedFile,
		"-f",
	).Wait(redirect.config.DefaultTimeoutDuration())
	Expect(downloadCurl).To(Exit(0))
}

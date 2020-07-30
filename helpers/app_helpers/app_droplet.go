package app_helpers

import (
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"

	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/config"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/download"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

type AppDroplet struct {
	guid string
	appGuid string
	config  config.CatsConfig
}

func NewAppDroplet(appGuid string, config config.CatsConfig) *AppDroplet {
	return &AppDroplet{
		appGuid: appGuid,
		config:  config,
	}
}

func CreateEmptyDroplet(appGuid string) *AppDroplet{
	var createDropletBody struct {
		Relationships struct {
			App struct {
				Data struct{
					Guid string `json:"guid"`
				} `json:"data"`
			} `json:"app"`
		} `json:"relationships"`
	}
	createDropletBody.Relationships.App.Data.Guid = appGuid
	jsonBody, err := json.Marshal(createDropletBody)
	Expect(err).NotTo(HaveOccurred())

	var dropletResponseJSON struct {
		Guid string `json:"guid"`
		Relationships struct {
			App struct {
				Data struct{
					Guid string `json:"guid"`
				} `json:"data"`
			} `json:"app"`
		} `json:"relationships"`
	}

	workflowhelpers.ApiRequest(
		"POST",
		"/v3/droplets",
		&dropletResponseJSON,
		Config.DefaultTimeoutDuration(),
		string(jsonBody),
	)

	Expect(dropletResponseJSON.Relationships.App.Data.Guid).To(Equal(appGuid))
	var newEmptyDroplet = AppDroplet{
		config: Config,
		guid: dropletResponseJSON.Guid,
		appGuid: dropletResponseJSON.Relationships.App.Data.Guid,
	}
	return &newEmptyDroplet
}

func (droplet *AppDroplet) DownloadTo(downloadPath string) (string, error) {
	dropletTarballPath := fmt.Sprintf("%s.tar.gz", downloadPath)
	dropletGuid := v3_helpers.GetCurrentDropletGuidFromApp(droplet.appGuid)
	downloadUrl := fmt.Sprintf("/v3/droplets/%s/download", dropletGuid)

	err := download.WithRedirect(downloadUrl, dropletTarballPath, droplet.config)
	return dropletTarballPath, err
}

func (droplet *AppDroplet) UploadFrom(uploadPath string) {
	token := v3_helpers.GetAuthToken()
	uploadURL := fmt.Sprintf("%s%s/v3/droplets/%s/upload", droplet.config.Protocol(), droplet.config.GetApiEndpoint(), droplet.guid)
	bits := fmt.Sprintf(`bits=@%s`, uploadPath)

	curl := helpers.CurlRedact(token, droplet.config, uploadURL, "-v", "-X", "POST", "-F", bits, "-H", fmt.Sprintf("Authorization: %s", token)).Wait()
	Expect(curl).To(Exit(0))

	var dropletResponseJSON struct {
		Guid string `json:"guid"`
		Relationships struct {
			App struct {
				Data struct{
					Guid string `json:"guid"`
				} `json:"data"`
			} `json:"app"`
		} `json:"relationships"`
	}
	bytes := curl.Out.Contents()
	json.Unmarshal(bytes, &dropletResponseJSON)
	Expect(dropletResponseJSON.Guid).NotTo(Equal(""))

	Eventually(func() *Session {
		return cf.Cf("curl", fmt.Sprintf("/v3/droplets/%s", droplet.guid)).Wait()
	}).Should(Say("STAGED"))
}

func (droplet *AppDroplet) SetAsCurrentDroplet() {
	appGuid := droplet.appGuid
	token := v3_helpers.GetAuthToken()

	currentDropletUrl :=  fmt.Sprintf("/v3/apps/%s/relationships/current_droplet", appGuid)
	curl := cf.Cf("curl", currentDropletUrl, "-X", "PATCH", "-d", fmt.Sprintf("'{ \"data\": { \"guid\": \"%s\" } }'", droplet.guid), "-H", fmt.Sprintf("Authorization: %s", token)).Wait()
	Expect(curl).To(Exit(0))

	session := cf.Cf("curl", currentDropletUrl)
	Eventually(session).Should(Say(droplet.guid))
}
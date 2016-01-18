package v3

import (
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("v3 tasks", func() {
	var (
		appName                         string
		appGuid                         string
		packageGuid                     string
		spaceGuid                       string
		appCreationEnvironmentVariables string
	)

	BeforeEach(func() {
		appName = generator.PrefixedRandomName("CATS-APP-")
		spaceGuid = GetSpaceGuidFromName(context.RegularUserContext().Space)
		appCreationEnvironmentVariables = `"foo"=>"bar"`
		appGuid = CreateApp(appName, spaceGuid, `{"foo":"bar"}`)
		packageGuid = CreatePackage(appGuid)
		token := GetAuthToken()
		uploadUrl := fmt.Sprintf("%s/v3/packages/%s/upload", config.ApiEndpoint, packageGuid)
		UploadPackage(uploadUrl, assets.NewAssets().DoraZip, token)
		WaitForPackageToBeReady(packageGuid)
		dropletGuid := StagePackage(packageGuid, "{}")
		WaitForDropletToStage(dropletGuid)
		AssignDropletToApp(appGuid, dropletGuid)
	})

	AfterEach(func() {
		DeleteApp(appGuid)
	})

	XIt("can create a task", func() {
		postBody := `{"command": "echo 0", "name": "mreow"}`
		createCommand := cf.Cf("curl", fmt.Sprintf("/v3/apps/%s/tasks", appGuid), "-X", "POST", "-d", postBody).Wait(DEFAULT_TIMEOUT)
		Expect(createCommand).To(Exit(0))
		Expect(createCommand.Out.Contents()).To(ContainSubstring("echo 0"))
		Expect(createCommand.Out.Contents()).To(ContainSubstring("mreow"))
		Expect(createCommand.Out.Contents()).To(ContainSubstring("RUNNING"))
	})
})

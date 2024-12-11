package file_based_service_bindings

import (
	"encoding/json"
	"fmt"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/services"
	"github.com/cloudfoundry/cf-test-helpers/v2/generator"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	"strings"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = FileBasedServiceBindingsBuildpackAppDescribe("File Based Service Bindings", func() {
	Describe("Enabling a file based service binding for a buildpack app", func() {
		var appName, serviceName string

		getEncodedFilepath := func(serviceName string, fileName string) string {
			path := fmt.Sprintf("/etc/cf-service-bindings/%s/%s", serviceName, fileName)
			return strings.Replace(path, "/", "%2F", -1)
		}

		checkFileContent := func(fileName string, content string) {
			curlResponse := helpers.CurlApp(Config, appName, "/file/"+getEncodedFilepath(serviceName, fileName))
			Expect(curlResponse).Should(ContainSubstring(content))
		}

		getServiceInstanceGuid := func(serviceName string) string {
			serviceGuidCmd := cf.Cf("service", serviceName, "--guid")
			Eventually(serviceGuidCmd).Should(Exit(0))
			return strings.TrimSpace(string(serviceGuidCmd.Out.Contents()))
		}

		getServiceBindingGuid := func(appGuid string, instanceGuid string) string {
			jsonResults := services_test.Response{}
			bindingCurl := cf.Cf("curl", fmt.Sprintf("/v3/service_credential_bindings?app_guids=%s&service_instance_guids=%s", appGuid, instanceGuid)).Wait()
			Expect(bindingCurl).To(Exit(0))
			Expect(json.Unmarshal(bindingCurl.Out.Contents(), &jsonResults)).NotTo(HaveOccurred())
			Expect(len(jsonResults.Resources)).To(BeNumerically(">", 0), "Expected to find at least one service binding.")
			return jsonResults.Resources[0].GUID
		}

		BeforeEach(func() {
			appName = random_name.CATSRandomName("APP")
			serviceName = generator.PrefixedRandomName("cats", "svin") // uppercase characters are not valid
		})

		AfterEach(func() {
			app_helpers.AppReport(appName)
			Eventually(cf.Cf("unbind-service", appName, serviceName).Wait()).Should(Exit(0))
			Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
			Eventually(cf.Cf("delete-service", serviceName, "-f").Wait()).Should(Exit(0))
		})

		FIt("creates the required files in the app container", func() {
			tags := "['list', 'of', 'tags']"
			creds := `{"username": "admin", "password":"pa55woRD"}`
			Expect(cf.Cf("create-user-provided-service", serviceName, "-p", creds, "-t", tags).Wait()).To(Exit(0))
            serviceGuid := getServiceInstanceGuid(serviceName)

			Expect(cf.Cf("create-app", appName).Wait()).To(Exit(0))
			appGuid := app_helpers.GetAppGuid(appName)

			appFeatureUrl := fmt.Sprintf("/v3/apps/%s/features/file-based-service-bindings", appGuid)
			Expect(cf.Cf("curl", appFeatureUrl, "-X", "PATCH", "-d", `{"enabled": true}`).Wait()).To(Exit(0))

			Expect(cf.Cf("bind-service", appName, serviceName).Wait()).To(Exit(0))

			Expect(cf.Cf(app_helpers.CatnipWithArgs(
				appName,
				"-m", DEFAULT_MEMORY_LIMIT)...,
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			checkFileContent("binding-guid", getServiceBindingGuid(appGuid, serviceGuid))
			checkFileContent("instance-guid", serviceGuid)
			checkFileContent("instance-name", serviceName)
			checkFileContent("label", "user-provided")
			checkFileContent("name", serviceName)
			checkFileContent("password", "pa55woRD")
			checkFileContent("provider", "user-provided")
			// TODO doesn't work yet: someone transforms the result to ["['list'","'of'","'tags']"]
			// checkFileContent("tags", `["list", "of", "tags"]`)
			checkFileContent("type", "user-provided")
			checkFileContent("username", "admin")
		})
	})
})

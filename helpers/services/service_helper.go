package services

import (
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

type LastOperation struct {
	State string `json:"state"`
}

type Resource struct {
	Name          string `json:"name"`
	GUID          string
	LastOperation LastOperation `json:"last_operation"`
}

type Response struct {
	Resources []Resource `json:"resources"`
}

type ErrorResponse struct {
	ErrorCode string `json:"error_code"`
}

func GetServiceInstanceGuid(serviceName string) string {
	serviceGuidCmd := cf.Cf("service", serviceName, "--guid")
	Eventually(serviceGuidCmd).Should(Exit(0))
	return strings.TrimSpace(string(serviceGuidCmd.Out.Contents()))
}

func GetServiceBindingGuid(appGuid string, instanceGuid string) string {
	jsonResults := Response{}
	bindingCurl := cf.Cf("curl", fmt.Sprintf("/v3/service_credential_bindings?app_guids=%s&service_instance_guids=%s", appGuid, instanceGuid)).Wait()
	Expect(bindingCurl).To(Exit(0))
	Expect(json.Unmarshal(bindingCurl.Out.Contents(), &jsonResults)).NotTo(HaveOccurred())
	Expect(len(jsonResults.Resources)).To(BeNumerically(">", 0), "Expected to find at least one service binding.")
	return jsonResults.Resources[0].GUID
}

func ValidateServiceBindingK8s(appName, serviceName, appGuid, serviceGuid string) {
	serviceBindingRoot := helpers.CurlApp(Config, appName, "/env/SERVICE_BINDING_ROOT")
	Expect(serviceBindingRoot).Should(Equal("/etc/cf-service-bindings"))

	getEncodedFilepath := func(serviceName string, fileName string) string {
		path := fmt.Sprintf("%s/%s/%s", serviceBindingRoot, serviceName, fileName)
		return strings.Replace(path, "/", "%2F", -1)
	}

	checkFileContent := func(fileName string, content string) {
		curlResponse := helpers.CurlApp(Config, appName, "/file/"+getEncodedFilepath(serviceName, fileName))
		Expect(curlResponse).Should(ContainSubstring(content))
	}

	checkFileContent("binding_guid", GetServiceBindingGuid(appGuid, serviceGuid))
	checkFileContent("instance_guid", serviceGuid)
	checkFileContent("instance_name", serviceName)
	checkFileContent("label", "user-provided")
	checkFileContent("name", serviceName)
	checkFileContent("password", "pa55woRD")
	checkFileContent("provider", "user-provided")
	checkFileContent("tags", `["list","of","tags"]`)
	checkFileContent("type", "user-provided")
	checkFileContent("username", "admin")
}

func ValidateFileBasedVcapServices(appName, serviceName, appGuid, serviceGuid string) {
	vcapServicesFilePath := helpers.CurlApp(Config, appName, "/env/VCAP_SERVICES_FILE_PATH")
	Expect(vcapServicesFilePath).Should(Equal("/etc/cf-service-bindings/vcap_services"))

	getEncodedFilepath := func() string {
		return strings.Replace(vcapServicesFilePath, "/", "%2F", -1)
	}

	expectedVcapServicesTemplate := `{
		"user-provided": [
			{
		      "label": "user-provided",
		      "name": "%s",
		      "tags": [
		       "list", "of", "tags"
		      ],
		      "instance_guid": "%s",
		      "instance_name": "%s",
		      "binding_guid": "%s",
		      "binding_name": "",
		      "credentials": {
		        "password": "pa55woRD",
		        "username": "admin"
		      }
			}
		]
    }`

	expectedString := fmt.Sprintf(expectedVcapServicesTemplate, serviceName, serviceGuid, serviceName, GetServiceBindingGuid(appGuid, serviceGuid))
	expectedJson := VCAPServicesFile{}
	err := expectedJson.ReadFromString(expectedString)
	if err != nil {
		Fail(err.Error())
	}

	curlResponse := helpers.CurlApp(Config, appName, "/file/"+getEncodedFilepath())
	actualJson := VCAPServicesFile{}
	err = actualJson.ReadFromString(curlResponse)
	if err != nil {
		Fail(err.Error())
	}

	Expect(actualJson).To(Equal(expectedJson))
}

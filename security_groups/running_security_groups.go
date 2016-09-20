package security_groups_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
)

var _ = SecurityGroupsDescribe("Security Groups", func() {

	type AppResource struct {
		Metadata struct {
			Url string
		}
	}
	type AppsResponse struct {
		Resources []AppResource
	}

	type Stat struct {
		Stats struct {
			Host string
			Port int
		}
	}
	type StatsResponse map[string]Stat

	type DoraCurlResponse struct {
		Stdout     string
		Stderr     string
		ReturnCode int `json:"return_code"`
	}

	var serverAppName, securityGroupName, privateHost string
	var privatePort int

	BeforeEach(func() {
		serverAppName = random_name.CATSRandomName("APP")
		Expect(cf.Cf("push",
			serverAppName,
			"--no-start",
			"-b", Config.RubyBuildpackName,
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().Dora,
			"-d", Config.AppsDomain).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		app_helpers.SetBackend(serverAppName)
		Expect(cf.Cf("start", serverAppName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		// gather app url
		var appsResponse AppsResponse
		cfResponse := cf.Cf("curl", fmt.Sprintf("/v2/apps?q=name:%s", serverAppName)).Wait(DEFAULT_TIMEOUT).Out.Contents()
		json.Unmarshal(cfResponse, &appsResponse)
		serverAppUrl := appsResponse.Resources[0].Metadata.Url

		// gather app stats for execution vm ip and app port
		var statsResponse StatsResponse
		cfResponse = cf.Cf("curl", fmt.Sprintf("%s/stats", serverAppUrl)).Wait(DEFAULT_TIMEOUT).Out.Contents()
		json.Unmarshal(cfResponse, &statsResponse)

		privateHost = statsResponse["0"].Stats.Host
		privatePort = statsResponse["0"].Stats.Port
	})

	AfterEach(func() {
		app_helpers.AppReport(serverAppName, DEFAULT_TIMEOUT)

		Expect(cf.Cf("delete", serverAppName, "-f", "-r").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
	})

	// This test assumes the default running security groups block access to the vms that execute apps.
	// The test takes advantage of the fact that the execution vm's ip address and internal container ip address
	//   are discoverable via the cc api and dora's myip endpoint.
	It("allows previously-blocked ip traffic after applying a security group, and re-blocks it when the group is removed", func() {

		clientAppName := random_name.CATSRandomName("APP")
		Expect(cf.Cf("push", clientAppName, "--no-start", "-b", Config.RubyBuildpackName, "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().Dora, "-d", Config.AppsDomain).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		app_helpers.SetBackend(clientAppName)
		Expect(cf.Cf("start", clientAppName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		defer func() { cf.Cf("delete", clientAppName, "-f", "-r").Wait(CF_PUSH_TIMEOUT) }()

		By("Gathering container ip")
		curlResponse := helpers.CurlApp(serverAppName, "/myip")
		containerIp := strings.TrimSpace(curlResponse)

		By("Gathering container port")
		curlResponse = helpers.CurlApp(serverAppName, "/env/VCAP_APPLICATION")
		var env map[string]interface{}
		err := json.Unmarshal([]byte(curlResponse), &env)
		Expect(err).NotTo(HaveOccurred())
		containerPort := int(env["port"].(float64))

		By("Testing app egress rules")
		var doraCurlResponse DoraCurlResponse
		curlResponse = helpers.CurlApp(clientAppName, fmt.Sprintf("/curl/%s/%d", privateHost, privatePort))
		json.Unmarshal([]byte(curlResponse), &doraCurlResponse)
		Expect(doraCurlResponse.ReturnCode).ToNot(Equal(0))

		By("Applying security group")
		rules := fmt.Sprintf(
			`[{"destination":"%s","ports":"%d","protocol":"tcp"},
			{"destination":"%s","ports":"%d","protocol":"tcp"}]`,
			privateHost, privatePort, containerIp, containerPort)

		file, _ := ioutil.TempFile(os.TempDir(), "CATS-sg-rules")
		defer os.Remove(file.Name())
		file.WriteString(rules)

		rulesPath := file.Name()
		securityGroupName = random_name.CATSRandomName("SG")

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			Expect(cf.Cf("create-security-group", securityGroupName, rulesPath).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			Expect(
				cf.Cf("bind-security-group",
					securityGroupName,
					TestSetup.RegularUserContext().Org,
					TestSetup.RegularUserContext().Space).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})
		defer func() {
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				Expect(cf.Cf("delete-security-group", securityGroupName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			})
		}()

		Expect(cf.Cf("restart", clientAppName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		By("Testing app egress rules")
		curlResponse = helpers.CurlApp(clientAppName, fmt.Sprintf("/curl/%s/%d", privateHost, privatePort))
		json.Unmarshal([]byte(curlResponse), &doraCurlResponse)
		Expect(doraCurlResponse.ReturnCode).To(Equal(0))

		By("Unapplying security group")
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			Expect(cf.Cf("unbind-security-group", securityGroupName, TestSetup.RegularUserContext().Org, TestSetup.RegularUserContext().Space).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})
		Expect(cf.Cf("restart", clientAppName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		By("Testing app egress rules")
		curlResponse = helpers.CurlApp(clientAppName, fmt.Sprintf("/curl/%s/%d", privateHost, privatePort))
		json.Unmarshal([]byte(curlResponse), &doraCurlResponse)
		Expect(doraCurlResponse.ReturnCode).ToNot(Equal(0))
	})

	It("allows external and denies internal traffic during staging based on default staging security rules", func() {
		buildpack := random_name.CATSRandomName("BPK")
		testAppName := random_name.CATSRandomName("APP")
		privateUri := fmt.Sprintf("%s:%d", privateHost, privatePort)

		buildpackZip := assets.NewAssets().SecurityGroupBuildpack

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			Expect(cf.Cf("create-buildpack", buildpack, buildpackZip, "999").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})
		defer func() {
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				Expect(cf.Cf("delete-buildpack", buildpack, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			})
		}()

		Expect(cf.Cf("push", testAppName, "--no-start", "-b", Config.RubyBuildpackName, "-m", DEFAULT_MEMORY_LIMIT, "-b", buildpack, "-p", assets.NewAssets().HelloWorld, "-d", Config.AppsDomain).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		app_helpers.SetBackend(testAppName)
		defer func() { cf.Cf("delete", testAppName, "-f", "-r").Wait(CF_PUSH_TIMEOUT) }()

		Expect(cf.Cf("set-env", testAppName, "TESTURI", "www.google.com").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		Expect(cf.Cf("start", testAppName).Wait(CF_PUSH_TIMEOUT)).To(Exit(1))
		Eventually(func() *Session {
			appLogsSession := cf.Cf("logs", "--recent", testAppName)
			Expect(appLogsSession.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			return appLogsSession
		}, 5).Should(Say("CURL_EXIT=0"))

		Expect(cf.Cf("set-env", testAppName, "TESTURI", privateUri).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		Expect(cf.Cf("restart", testAppName).Wait(CF_PUSH_TIMEOUT)).To(Exit(1))
		Eventually(func() *Session {
			appLogsSession := cf.Cf("logs", "--recent", testAppName)
			Expect(appLogsSession.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			return appLogsSession
		}, 5).Should(Say("CURL_EXIT=[^0]"))
	})
})

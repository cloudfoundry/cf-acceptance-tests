package windows

import (
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = WindowsDescribe("WCF", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")

		Expect(cf.Cf("push",
			appName,
			"--no-start",
			"-s", Config.GetWindowsStack(),
			"-b", Config.GetHwcBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().Wcf,
			"-i", strconv.Itoa(Config.GetNumWindowsCells()+1),
			"-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		app_helpers.SetBackend(appName)
		Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).Should(Exit(0))
	})

	It("can have multiple routable instances on the same cell", func() {
		Eventually(allInstancesRunning(appName, Config.GetNumWindowsCells()+1), Config.CfPushTimeoutDuration()).Should(Succeed())

		Expect(wcfRequest(appName).Msg).To(Equal("WATS!!!"))

		Eventually(isServiceRunningOnTheSameCell(appName), Config.CfPushTimeoutDuration()).Should(BeTrue())
	})
})

func allInstancesRunning(appName string, instances int) func() error {
	return func() error {
		type StatsResponse map[string]struct {
			State string `json:"state"`
		}

		session := cf.Cf("app", appName, "--guid")
		Expect(session.Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

		appGuid := strings.Replace(string(session.Out.Contents()), "\n", "", -1)

		endpoint := fmt.Sprintf("/v2/apps/%s/stats", appGuid)

		var response StatsResponse
		ApiRequest("GET", endpoint, &response, Config.DefaultTimeoutDuration())

		var err error
		for k, v := range response {
			if v.State != "RUNNING" {
				err = errors.New(fmt.Sprintf("App %s instance %s is not running: State = %s", appName, k, v.State))
			}
		}
		return err
	}
}

type WCFResponse struct {
	Msg          string
	InstanceGuid string
	CFInstanceIp string
}

func wcfRequest(appName string) WCFResponse {
	uri := helpers.AppUri(appName, "/Hello.svc?wsdl", Config)

	helloMsg := `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"><s:Body><Echo xmlns="http://tempuri.org/"><msg>WATS!!!</msg></Echo></s:Body></s:Envelope>`
	buf := strings.NewReader(helloMsg)
	req, err := http.NewRequest("POST", uri, buf)
	req.Header.Add("Content-Type", "text/xml")
	req.Header.Add("SOAPAction", "http://tempuri.org/IHelloService/Echo")
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	resp, err := client.Do(req)
	Expect(err).To(BeNil())
	defer resp.Body.Close()

	xmlDecoder := xml.NewDecoder(resp.Body)
	type SoapResponse struct {
		XMLResult string `xml:"Body>EchoResponse>EchoResult"`
	}
	xmlResponse := SoapResponse{}
	Expect(xmlDecoder.Decode(&xmlResponse)).To(BeNil())
	results := strings.Split(xmlResponse.XMLResult, ",")
	Expect(len(results)).To(Equal(3))
	return WCFResponse{
		Msg:          results[0],
		CFInstanceIp: results[1],
		InstanceGuid: results[2],
	}
}

func isServiceRunningOnTheSameCell(appName string) bool {
	// Keep track of the IDs of the instances we have reached
	output := map[string]string{}
	for i := 0; i < Config.GetNumWindowsCells()*5; i++ {
		res := wcfRequest(appName)
		guids := output[res.CFInstanceIp]
		if guids != "" && !strings.Contains(guids, res.InstanceGuid) {
			return true
		}
		output[res.CFInstanceIp] = res.InstanceGuid
	}
	return false
}

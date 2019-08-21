package windows

import (
	"crypto/tls"
	"encoding/xml"
	"net/http"
	"strings"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
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
			"-s", Config.GetWindowsStack(),
			"-b", Config.GetHwcBuildpackName(),
			"-m", DEFAULT_WINDOWS_MEMORY_LIMIT,
			"-p", assets.NewAssets().Wcf,
			"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).Should(Exit(0))
	})

	It("can push a WCF app", func() {
		Expect(wcfRequest(appName).Msg).To(Equal("WATS!!!"))
	})
})

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

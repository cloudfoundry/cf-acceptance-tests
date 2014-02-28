package services

import (
	"fmt"
	"testing"
	"encoding/json"
	"path/filepath"
	"io/ioutil"
	"strconv"
	"os"

	. "github.com/onsi/ginkgo"
	ginkgoconfig "github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/vito/cmdtest"
	. "github.com/vito/cmdtest/matchers"

	"../config"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/runner"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
)

type ServiceBrokerConfigData struct {
	FirstBrokerServiceLabel string `json:"first_broker_service_label"`
	FirstBrokerPlanName     string `json:"first_broker_plan_name"`
	SecondBrokerServiceLabel string `json:"second_broker_service_label"`
	SecondBrokerPlanName     string `json:"second_broker_plan_name"`
}

var IntegrationConfig = config.Load()
var serviceBrokerPath string
var ServiceBrokerConfigPath string
var ServiceBrokerConfig ServiceBrokerConfigData
var homePath string

func TestServices(t *testing.T) {
	RegisterFailHandler(Fail)

	LoadServiceBrokerConfig()
	CreateHomeConfig()
	RunSpecsWithDefaultAndCustomReporters(t, "Services", []Reporter{reporters.NewJUnitReporter(fmt.Sprintf("junit_%d.xml", ginkgoconfig.GinkgoConfig.ParallelNode))})
	RemoveHomeConfig()
}

func AppUri(appName string, endpoint string) string {
	return "http://" + appName + "." + IntegrationConfig.AppsDomain + endpoint
}

func Curling(args ...string) func() *cmdtest.Session {
	return func() *cmdtest.Session {
		return Curl(args...)
	}
}

func LoadServiceBrokerConfig() {
	serviceBrokerPath = "../assets/service_broker"
	ServiceBrokerConfigPath, _ = filepath.Abs("./config.json")
	ServiceBrokerConfig = ServiceBrokerConfigData{}
	configJSON, _ := ioutil.ReadFile(ServiceBrokerConfigPath)
	json.Unmarshal(configJSON, &ServiceBrokerConfig)
}

func CreateHomeConfig() {
	homePath = fmt.Sprintf("%s/cf_config_%s", os.Getenv("HOME"), strconv.Itoa(ginkgoconfig.GinkgoConfig.ParallelNode))
	os.MkdirAll(homePath, os.ModePerm)
	os.Setenv("CF_HOME", homePath)

	Expect(Cf("api", os.Getenv("API_ENDPOINT"))).To(ExitWith(0))
	Expect(Cf("login", "-u", os.Getenv("CF_USER"), "-p", os.Getenv("CF_USER_PASSWORD"), "-o", os.Getenv("CF_ORG"), "-s", os.Getenv("CF_SPACE"))).To(ExitWith(0))
}

func RemoveHomeConfig() {
	os.RemoveAll(homePath)
}

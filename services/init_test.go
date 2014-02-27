package services

import (
	"fmt"
	"testing"
	"encoding/json"
	"path/filepath"
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	ginkgoconfig "github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/vito/cmdtest"

	"../config"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/runner"
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

func TestServices(t *testing.T) {
	serviceBrokerPath = "../assets/service_broker"
	ServiceBrokerConfigPath, _ = filepath.Abs("./config.json")
	ServiceBrokerConfig = ServiceBrokerConfigData{}
	configJSON, _ := ioutil.ReadFile(ServiceBrokerConfigPath)
	json.Unmarshal(configJSON, &ServiceBrokerConfig)

	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "Services", []Reporter{reporters.NewJUnitReporter(fmt.Sprintf("junit_%d.xml", ginkgoconfig.GinkgoConfig.ParallelNode))})
}

func AppUri(appName string, endpoint string) string {
	return "http://" + appName + "." + IntegrationConfig.AppsDomain + endpoint
}

func Curling(args ...string) func() *cmdtest.Session {
	return func() *cmdtest.Session {
		return Curl(args...)
	}
}

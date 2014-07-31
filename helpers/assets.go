package helpers

type Assets struct {
	Dora                     string
	HelloWorld               string
	Node                     string
	Java                     string
	Golang                   string
	Python                   string
	LoggregatorLoadGenerator string
	ServiceBroker            string
	Php                      string
	SecurityGroupBuildpack   string
}

func NewAssets() Assets {
	return Assets{
		Dora:       "../assets/dora",
		HelloWorld: "../assets/hello-world",
		Node:       "../assets/node",
		Java:       "../assets/java",
		Golang:     "../assets/golang",
		Python:     "../assets/python",
		LoggregatorLoadGenerator: "../assets/loggregator-load-generator",
		ServiceBroker:            "../assets/service_broker",
		Php:                      "../assets/php",
		SecurityGroupBuildpack: "../assets/security_group_buildpack",
	}
}

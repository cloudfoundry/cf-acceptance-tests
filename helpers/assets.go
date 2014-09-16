package helpers

type Assets struct {
	Dora                     string
	HelloWorld               string
	Node                     string
	NodeWithProcfile         string
	Java                     string
	Golang                   string
	Standalone               string
	Python                   string
	LoggregatorLoadGenerator string
	ServiceBroker            string
	Php                      string
	SecurityGroupBuildpack   string
	Fuse                     string
}

func NewAssets() Assets {
	return Assets{
		Dora:             "../assets/dora",
		HelloWorld:       "../assets/hello-world",
		Node:             "../assets/node",
		NodeWithProcfile: "../assets/node-with-procfile",
		Java:             "../assets/java",
		Golang:           "../assets/golang",
		Standalone:       "../assets/standalone",
		Python:           "../assets/python",
		LoggregatorLoadGenerator: "../assets/loggregator-load-generator",
		ServiceBroker:            "../assets/service_broker",
		Php:                      "../assets/php",
		SecurityGroupBuildpack: "../assets/security_group_buildpack.zip",
		Fuse: "../assets/fuse-mount",
	}
}

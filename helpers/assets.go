package helpers

type Assets struct {
	Dora                     string
	HelloWorld               string
	Node                     string
	Java                     string
	Go                       string
	Python                   string
	LoggregatorLoadGenerator string
	ServiceBroker            string
}

func NewAssets() Assets {
	return Assets{
		Dora:       "../assets/dora",
		HelloWorld: "../assets/hello-world",
		Node:       "../assets/node",
		Java:       "../assets/java",
		Go:         "../assets/go",
		Python:     "../assets/python",
		LoggregatorLoadGenerator: "../assets/loggregator-load-generator",
		ServiceBroker:            "../assets/service_broker",
	}
}

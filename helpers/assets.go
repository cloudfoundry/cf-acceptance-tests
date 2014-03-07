package helpers

type Assets struct {
	Dora string
	HelloWorld string
	LoggregatorLoadGenerator string
	ServiceBroker string
}

func NewAssets() Assets {
	return Assets{
		Dora: "../assets/dora",
		HelloWorld: "../assets/hello-world",
		LoggregatorLoadGenerator: "../assets/loggregator-load-generator",
		ServiceBroker: "../assets/service_broker",
	}
}

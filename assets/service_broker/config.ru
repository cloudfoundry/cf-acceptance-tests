$: << File.expand_path("../.", __FILE__)

require "service_broker"

use Rack::RewindableInput::Middleware

run ServiceBroker

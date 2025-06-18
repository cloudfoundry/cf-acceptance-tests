require 'sinatra'
require 'net/http'
require 'logger'

STDOUT.sync = true
STDERR.sync = true

ENDPOINT_TYPE_MAP = {
  :'api.ipify.org' => {
    path: '/ipv4-test'
  },
  :'api6.ipify.org' => {
    path: '/ipv6-test'
  },
  :'api64.ipify.org' => {
    path: '/dual-stack-test'
  }
}.freeze

def logger
  @logger ||= Logger.new($stdout).tap do |log|
    log.progname = 'SinatraApp'
    log.level = Logger::INFO
  end
end

class IPTester
  def initialize(endpoint)
    @endpoint = endpoint
  end

  def fetch_ip
    uri = URI("http://#{@endpoint}/")

    begin
      response = Net::HTTP.get_response(uri)
      response.body.strip
    rescue => e
      logger.error("Failed to reach #{@endpoint}: #{e.class} - #{e.message}")
      "Error fetching IP: #{e.message}"
    end
  end
end

configure do
  set :port, ENV.fetch('PORT', 8080).to_i
  set :bind, ENV.fetch('VCAP_APP_HOST', '127.0.0.1')
end

ENDPOINT_TYPE_MAP.each do |endpoint, data|
  get data[:path] do
    tester = IPTester.new(endpoint)
    message = tester.fetch_ip
    "#{message}"
  end
end

get '/' do
<<-RESPONSE
Healthy\n
It just needed to be restarted!\n
My application metadata: #{ENV['VCAP_APPLICATION']}\n
My port: #{ENV['PORT']}\n
My custom env variable: #{ENV['CUSTOM_VAR']}\n
RESPONSE
end

get '/log/:message' do
  message = params[:message]
  STDOUT.puts(message)
  "logged #{message} to STDOUT"
end

Thread.new do
  while true do
    STDOUT.puts "Tick: #{Time.now.to_i}"
    sleep 1
  end
end
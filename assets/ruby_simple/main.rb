require 'sinatra'
require 'net/http'
require 'ipaddr'
require 'logger'

STDOUT.sync = true
STDERR.sync = true

ENDPOINT_TYPE_MAP = {
  :'api.ipify.org' => {
    validation_name: 'IPv4',
    path: '/ipv4-test'
  },
  :'api6.ipify.org' => {
    validation_name: 'IPv6',
    path: '/ipv6-test'
  },
  :'api64.ipify.org' => {
    validation_name: 'Dual stack',
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

  def test_single_address
    test_endpoint.tap do |result|
      print_result(result)
    end
  end

  private

  def print_result(result)
    validation_type = ENDPOINT_TYPE_MAP[@endpoint][:validation_name]
    message = "#{validation_type} validation #{result[:success] ? 'succeeded' : 'failed'}."
    result[:success] ? logger.info(message) : logger.error(message)
  end

  def test_endpoint
    logger.info("Testing endpoint: #{@endpoint}")
    uri = URI("http://#{@endpoint}/")

    begin
      response = Net::HTTP.get_response(uri)
      ip_type = determine_ip_type(response.body.strip)

      {
        success: response.is_a?(Net::HTTPSuccess),
        ip_type: ip_type
      }
    rescue => e
      logger.error("Failed to reach #{@endpoint}: #{e.class} - #{e.message}\n#{e.backtrace.join("\n")}")
      {
        success: false,
        error: e.message,
        ip_type: 'Unknown'
      }
    end
  end

  def determine_ip_type(ip_string)
    ip = IPAddr.new(ip_string)
    return 'IPv4' if ip.ipv4?
    return 'IPv6' if ip.ipv6?
    'Unknown'
  rescue IPAddr::InvalidAddressError
    'Invalid IP'
  end
end

configure do
  set :port, ENV.fetch('PORT', 8080).to_i
  set :bind, ENV.fetch('VCAP_APP_HOST', '127.0.0.1')
end

ENDPOINT_TYPE_MAP.each do |endpoint, data|
  get data[:path] do
    tester = IPTester.new(endpoint)
    result = tester.test_single_address
    status(result[:success] ? 200 : 500)

    validation_name = ENDPOINT_TYPE_MAP[endpoint][:validation_name]
    message = "#{validation_name} validation resulted in #{result[:success] ? 'success' : 'failure'}. Detected IP type is #{result[:ip_type]}."
    message += " Error message: #{result[:error]}" if result[:error]
    message
  end
end

get '/' do
<<-RESPONSE
Healthy
It just needed to be restarted!
My application metadata: #{ENV['VCAP_APPLICATION']}
My port: #{ENV['PORT']}
My custom env variable: #{ENV['CUSTOM_VAR']}
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
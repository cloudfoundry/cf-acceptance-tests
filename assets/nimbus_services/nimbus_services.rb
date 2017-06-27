require 'sinatra/base'
require 'json'
require 'httpclient'

$stdout.sync = true
$stderr.sync = true

module Nimbus
  class ServicesApp < Sinatra::Base

    get '/' do
      'OK'
    end

    get '/currtime' do
      cache_control :public, max_age: 60
      "#{Time.new.to_f * 1000}"
    end

    get '/nbconfig/test' do
      should_work     = "https://#{Config.application_uris.first}/"
      should_not_work = "https://www.bbc.co.uk/"

      status_should_work     = make_http_call(should_work)
      status_should_not_work = make_http_call(should_not_work)

      status_should_work == 'OK' && status_should_not_work == 'FIREWALLED' ? 'OK' : 'FAIL'
    end

    private

    def make_http_call(url)
      puts "about to call: #{url}"
      client = HTTPClient.new
      client.connect_timeout = 2

      response = client.get(url)
      response.status_code == 200 ? 'OK' : 'FAIL'
    rescue HTTPClient::ConnectTimeoutError
      puts "call to #{url} is firewalled"
      'FIREWALLED'
    end

  end

  class Config
    class << self

      def rabbit
        rabbit_entry = vcap_services.select { |key, _| key.include? 'rabbitmq' }.values[0][0]
        rabbit_entry['credentials']['uri']
      rescue
        nil
      end

      def redis
        redis_entry = vcap_services.select { |key, _| key.include? 'redis' }.values[0][0]
        credentials = redis_entry['credentials']
        "redis://#{credentials['name']}:#{credentials['password']}@#{credentials['host']}:#{credentials['port']}/0/nimbus:store"
      rescue
        nil
      end

      def mongo
        mongo_entry = vcap_services.select { |key, _| key.include? 'mongo' }.values[0][0]
        mongo_entry['credentials']
      rescue
        nil
      end

      def postgres
        postgres_entry = vcap_services.select { |key, _| key.include? 'postgresql' }.values[0][0]
        postgres_entry['credentials']
      rescue
        nil
      end

      def proxy
        proxy_entry = vcap_services.select { |key, _| key.include? 'proxy' }.values[0][0]
        proxy_entry['credentials']['http_proxy']
      rescue
        nil
      end

      def instance_index
        vcap_application['instance_index']
      end

      def vcap_services
        @vcap_services ||= JSON.parse(ENV['VCAP_SERVICES'])
      end

      def vcap_application
        @vcap_application ||= JSON.parse(ENV['VCAP_APPLICATION'])
      end

      def application_uris
        vcap_application['application_uris']
      end

    end
  end


end

require 'sinatra/base'
require 'httpclient'

module Nimbus
  class ServicesApp < Sinatra::Base

    configure do
      ENV['HTTP_PROXY'] = Nimbus::Config.internalproxy
    end

    get '/internalproxy' do
      response = HTTPClient.new.get 'http://nexus.bskyb.com'
      response.status_code == 200 ? 'OK' : 'FAIL'
    end

    private

    # makes sure WEB_PROXY_* env vars are set correctly
#    def proxy_env_vars_ok?
#      ENV['HTTP_PROXY'] == "http://#{ENV['WEB_PROXY_USER']}:#{ENV['WEB_PROXY_PASS']}@#{ENV['WEB_PROXY_HOST']}:#{ENV['WEB_PROXY_PORT']}"
#    end

  end
end


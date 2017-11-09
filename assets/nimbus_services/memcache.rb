require 'sinatra/base'
require 'dalli'

module Nimbus
  class ServicesApp < Sinatra::Base

    get '/memcache/insert/:key/:value' do
      memcache_client.set(params['key'], params['value'])
      'OK'
    end

    get '/memcache/read/:key/:value' do
      memcache_client.get(params['key']) == params['value'] ? 'OK' : 'FAIL'
    end

    private

    def memcache_client
      conf = Nimbus::Config.memcache
      url = "#{conf['host']}:#{conf['port']}"
      Dalli::Client.new(url, namespace: conf['name'])
    end

  end
end
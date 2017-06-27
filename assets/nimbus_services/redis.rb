require 'sinatra/base'
require 'redis'

module Nimbus
  class ServicesApp < Sinatra::Base

    get '/redis/insert/:key/:value' do
      redis = Redis.new(:url => Nimbus::Config.redis)
      redis.set(params['key'], params['value'])
      'OK'
    end

    get '/redis/read/:key/:value' do
      redis = Redis.new(:url => Nimbus::Config.redis)
      redis.get(params['key']) == params['value'] ? 'OK' : 'FAIL'
    end

  end
end
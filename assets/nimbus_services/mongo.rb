require 'active_model/serializers'
require 'mongo_mapper'
require 'sinatra/base'

module Nimbus
  class ServicesApp < Sinatra::Base

    configure do
      mongo_config = Nimbus::Config.mongo
      # does not handle multiple replica set hots
      # MongoMapper.setup({'production' => {'uri' => Nimbus::Config.mongo}}, 'production')

      hosts = mongo_config['hosts'].split(',')
      MongoMapper.connection = Mongo::MongoReplicaSetClient.new(
          hosts,
          :read => :secondary
      )
      MongoMapper.database = mongo_config['database']
      MongoMapper.database.authenticate(mongo_config['user'], mongo_config['password'])
    end

    get '/mongo/insert/:value' do
      MongoTest.new(value: params['value']).save!
      'OK'
    end

    get '/mongo/read/:value' do
      result = MongoTest.where(value: params['value'])
      result.size == 1 ? 'OK' : 'FAIL'
    end

  end

  class MongoTest
    include MongoMapper::Document

    key :value, String
  end

end
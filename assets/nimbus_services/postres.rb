require 'pg'
require 'sinatra/base'
require 'sinatra/activerecord'

module Nimbus
  class ServicesApp < Sinatra::Base

    register Sinatra::ActiveRecordExtension

    configure do
      set :database, adapter: 'postgresql',
                     database: Nimbus::Config.postgres['name'],
                     host: Nimbus::Config.postgres['host'],
                     port: Nimbus::Config.postgres['port'],
                     username: Nimbus::Config.postgres['username'],
                     password: Nimbus::Config.postgres['password']

      ActiveRecord::Base.logger = nil

      unless ActiveRecord::Base.connection.data_source_exists?(:test)
        puts 'creating test table'
        ActiveRecord::Base.connection.create_table :test do |t|
          t.string :value
        end
      end
    end

    get '/postgres/insert/:value' do
      value = params['value']
      PostgresTest.new(value: value).save
      'OK'
    end

    get '/postgres/read/:value' do
      result = []
      PostgresTest.uncached do
        result = PostgresTest.where(value: params['value'])
      end
      result.size == 1 ? 'OK' : 'FAIL'
    end

  end

  class PostgresTest < ActiveRecord::Base
    self.table_name = 'test'
  end

end
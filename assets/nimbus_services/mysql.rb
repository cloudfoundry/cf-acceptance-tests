require 'mysql2'
require 'sinatra/base'
require 'sinatra/activerecord'

module Nimbus
  class ServicesApp < Sinatra::Base

    register Sinatra::ActiveRecordExtension

    configure do
      set :database, adapter: 'mysql2',
                     database: Nimbus::Config.mysql['name'],
                     host: Nimbus::Config.mysql['host'],
                     port: Nimbus::Config.mysql['port'],
                     username: Nimbus::Config.mysql['username'],
                     password: Nimbus::Config.mysql['password']

      ActiveRecord::Base.logger = nil

      unless ActiveRecord::Base.connection.table_exists?(:test)
        puts 'creating test table'
        ActiveRecord::Base.connection.create_table :test do |t|
          t.string :value
        end
      end
    end

    get '/mysql/insert/:value' do
      value = params['value']
      MysqlTest.new(value: value).save
      'OK'
    end

    get '/mysql/read/:value' do
      result = []
      MysqlTest.uncached do
        result = MysqlTest.where(value: params['value'])
      end
      result.size == 1 ? 'OK' : 'FAIL'
    end

  end

  class MysqlTest < ActiveRecord::Base
    self.table_name = 'test'
  end

end
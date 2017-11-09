require 'sinatra/base'
require 'cassandra'

module Nimbus
  class ServicesApp < Sinatra::Base

    before do
      cql = <<-CQL
        CREATE TABLE IF NOT EXISTS test (
        key VARCHAR,
        val VARCHAR,
        PRIMARY KEY (key)
      )
      CQL
      cassandra_session.execute(cql)
    end

    after do
      @cassandra_session.close if @cassandra_session
    end

    get '/cassandra/insert/:key/:value' do |key, value|
      cql = <<-CQL
        INSERT INTO test (key, val) VALUES ('#{key}', '#{value}')
      CQL
      cassandra_session.execute(cql)
      'OK'
    end

    get '/cassandra/read/:key/:value' do |key, value|
      cql = <<-CQL
        SELECT val FROM test WHERE key = '#{key}'
      CQL
      result = cassandra_session.execute(cql)
      result.first['val'] == value ? 'OK' : 'FAIL'
    end

    private

    def cassandra_session

      unless @cassandra_session
        cassandra_cluster = Cassandra.cluster(
            username: Nimbus::Config.cassandra['username'],
            password: Nimbus::Config.cassandra['password'],
            hosts: Nimbus::Config.cassandra['hosts'].split(',')
        )

        @cassandra_session = cassandra_cluster.connect(Nimbus::Config.cassandra['keyspace'])
      end
      @cassandra_session
    end

  end
end

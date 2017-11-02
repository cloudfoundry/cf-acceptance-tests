require 'sinatra/base'
require 'amqp'
require 'pg'
require 'sinatra/activerecord'

QUEUE_NAME = 'nimbus.rabbit.test.queue'

# listen on the queue
EM.schedule do

  # connection option :heartbeat => 30 ?
  # would need to change string url to hash
  AMQP.connect(Nimbus::Config.rabbit) do |connection|
    # connection.on_tcp_connection_loss do |conn, settings|
    #   puts '[network failure] Trying to reconnect...'
    #   conn.reconnect(false, 2)
    # end

    channel = AMQP::Channel.new(connection, 2, :auto_recovery => true)
    exchange = channel.fanout(QUEUE_NAME)

    queue = channel.queue("", exclusive: true, auto_delete: true).bind(exchange)
    queue.subscribe do |payload|
      begin
        puts "message received: #{payload}"
        # store state in postgres
        Nimbus::RabbitMessageCounter.first.increment!(:value)
      rescue => ex
        puts "error processing message: #{ex.inspect}"
      end
    end

  end
end

module Nimbus

  class RabbitMessageCounter < ActiveRecord::Base
    self.table_name = 'test'
  end

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
          t.integer :value
        end
        puts 'inserting counter record'
        Nimbus::RabbitMessageCounter.new(value: 0).save
      end
    end

    get '/rabbit/publish' do
      AMQP.connect(Nimbus::Config.rabbit) do |connection|
        AMQP::Channel.new(connection).fanout(QUEUE_NAME) do |exchange|
          exchange.publish('message')
          connection.close
        end
      end
      puts 'message published to rabbitmq'
      'message published to rabbitmq'
    end

    get '/rabbit/check/:instance_count' do
      instance_count = params['instance_count'].to_i
      counter = Nimbus::RabbitMessageCounter.first.value
      record = Nimbus::RabbitMessageCounter.first
      record.value = 0
      record.save
      (counter == instance_count) ? 'OK' : "FAIL_EXPECTED_#{instance_count}_GOT_#{counter}"
    end

  end

end
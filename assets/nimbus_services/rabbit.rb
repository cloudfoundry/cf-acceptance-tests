require 'amqp'
require 'sinatra/base'
require 'redis'

QUEUE_NAME = 'nimbus.rabbit.test.queue'

# listen on the queue
EM.schedule do

  # connection option :heartbeat => 30 ?
  # would need to change string url to hash  
  AMQP.connect(Nimbus::Config.rabbit) do |connection|
    connection.on_tcp_connection_loss do |conn, settings|
      puts '[network failure] Trying to reconnect...'
      conn.reconnect(false, 2)
    end

    channel = AMQP::Channel.new(connection, 2, :auto_recovery => true)
    exchange = channel.fanout(QUEUE_NAME)

    queue = channel.queue("", exclusive: true, auto_delete: true).bind(exchange)
    queue.subscribe do |payload|
      begin
        puts "message received: #{payload}"
        # store state in redis
        redis = Redis.new(:url => Nimbus::Config.redis)
        redis.incr('counter')
      rescue => ex
        puts "error processing message: #{ex.inspect}"
      end
    end

  end
end

module Nimbus
  class ServicesApp < Sinatra::Base

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
      ok = false
      instance_count = params['instance_count'].to_i
      redis = Redis.new(:url => Nimbus::Config.redis)
      counter = redis.get('counter').to_i
      puts 'Resetting counter to zero'
      redis.set('counter', 0)
      if counter == instance_count
        ok = true
      end
      ok ? 'OK' : "FAIL_EXPECTED_#{instance_count}_GOT_#{counter}"
    end

  end
end
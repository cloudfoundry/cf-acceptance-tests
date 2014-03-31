ENV['RACK_ENV'] ||= 'development'

require 'rubygems'
require 'sinatra/base'
require 'json'
require 'pp'

ID = ((ENV["VCAP_APPLICATION"] && JSON.parse(ENV["VCAP_APPLICATION"])["instance_id"]) || SecureRandom.uuid).freeze

require 'bundler'
Bundler.require :default, ENV['RACK_ENV'].to_sym

CONFIG_DATA = ENV['CONFIG'] ? JSON.parse(ENV['CONFIG']) : { 'service' => {}, 'plan' => {}, 'dashboard_client' => {} }

$stdout.sync = true
$stderr.sync = true

class ServiceBroker < Sinatra::Base

  @@catalog = {
    'services' => [
      {
        'name' => 'fake-service',
        'id' => 'f479b64b-7c25-42e6-8d8f-e6d22c456c9b',
        'description' => 'fake service',
        'tags' => ['no-sql', 'relational'],
        'max_db_per_node' => 5,
        'bindable' => true,
        'metadata' => {
          'provider' => { 'name' => 'The name' },
          'listing' => {
            'imageUrl' => 'http://catgifpage.com/cat.gif',
            'blurb' => 'fake broker that is fake',
            'longDescription' => 'A long time ago, in a galaxy far far away...'
          },
          'displayName' => 'The Fake Broker'
        },
        'dashboard_client' => {
          'id'           => 'sso-test',
          'secret'       => 'sso-secret',
          'redirect_uri' => 'http://localhost:5551'
        }.merge(CONFIG_DATA['dashboard_client']),
        'plans' => [
          {
            'name' => 'fake-plan',
            'id' => 'f52eabf8-e38d-422f-8ef9-9dc83b75cc05',
            'description' => 'Shared fake Server, 5tb persistent disk, 40 max concurrent connections',
            'max_storage_tb' => 5,
            'metadata' => {
              'cost' => 0.0,
              'bullets' => [
                { 'content' => 'Shared fake server' },
                { 'content' => '5 TB storage' },
                { 'content' => '40 concurrent connections' }
              ]
            }
          }.merge(CONFIG_DATA['plan'])
        ]
      }.merge(CONFIG_DATA['service'])
    ]
  }

  get '/v2/catalog' do
    @@catalog.to_json
  end

  put '/v2/service_instances/:id' do
    status 201
    {}.to_json
  end

  delete '/v2/service_instances/:id' do
    status 200
    {}.to_json
  end

  get '/env/:name' do
    ENV[params[:name]]
  end

  get '/env' do
    ENV.to_hash.to_s
  end

  run! if app_file == $0
end

ENV['RACK_ENV'] ||= 'development'

require 'rubygems'
require 'sinatra/base'
require 'json'
require 'pp'

ID = ((ENV["VCAP_APPLICATION"] && JSON.parse(ENV["VCAP_APPLICATION"])["instance_id"]) || SecureRandom.uuid).freeze

require 'bundler'
Bundler.require :default, ENV['RACK_ENV'].to_sym

CONFIG_DATA = ENV['CONFIG'] ? JSON.parse(ENV['CONFIG']) : {}
SERVICE_ID  = '3fedb389-47ca-4102-9c1d-92308c5e1f8f'
PLAN_ID     = 'e190e178-21a3-4c1c-a262-b72cb8e50bbb'

$stdout.sync = true
$stderr.sync = true

class ServiceBroker < Sinatra::Base

  @@catalog_1 = {
    'services' => [
      {
        'name'=> CONFIG_DATA['first_broker_service_label'],
        'id'=> SERVICE_ID,
        'description'=>'fake service',
        'tags'=>['no-sql', 'relational'],
        'max_db_per_node'=>5,
        'bindable' => true,
        'metadata' => {
          'provider'=> {'name'=>'The name'},
          'listing' => {
            'imageUrl'=>'http://catgifpage.com/cat.gif',
            'blurb' => 'fake broker that is fake',
            'longDescription' => 'A long time ago, in a galaxy far far away...'
          },
          'displayName' => 'The Fake Broker'
        },
        'plans' => [
          {
            'name'=> CONFIG_DATA['first_broker_plan_name'],
            'id'=>PLAN_ID, #randomize
            'description'=>'Shared fake Server, 5tb persistent disk, 40 max concurrent connections',
            'max_storage_tb'=>5,
            'metadata' => {
              'cost'=>0.0,
              'bullets' => [
                {'content'=>'Shared fake server'},
                {'content'=>'5 TB storage'},
                {'content'=>'40 concurrent connections'}
              ]
            }
          }
        ]
      }
    ]
  }

  @@catalog_2 = {
    'services' => [
      {
        'name'=>CONFIG_DATA['second_broker_service_label'],
        'id'=> SERVICE_ID,
        'description'=>'fake service modified',
        'tags'=>['no-sql', 'relational'],
        'max_db_per_node'=>6,
        'bindable' => true,
        'metadata' => {
          'provider'=> {'name'=>'The new name'},
          'listing' => {
            'imageUrl'=>'http://catgifpage.com/cat.gif',
            'blurb' => 'new fake broker that is fake',
            'longDescription' => 'A long time ago, in a galaxy far far away...'
          },
          'displayName' => 'The new Fake Broker'
        },
        'plans' => [
          {
            'name'=>CONFIG_DATA['second_broker_plan_name'],
            'id'=>PLAN_ID, #randomize
            'description'=>'Shared fake Server, 6tb persistent disk, 41 max concurrent connections',
            'max_storage_tb'=>6,
            'metadata' => {
              'cost'=>0.0,
              'bullets' => [
                {'content'=>'Shared fake server'},
                {'content'=>'6 TB storage'},
                {'content'=>'41 concurrent connections'}
              ]
            }
          }
        ]
      }
    ]
  }

  @@current_catalog = @@catalog_1

  get '/v2/catalog' do
    @@current_catalog.to_json
  end

  post '/v2/catalog' do
    if @@current_catalog == @@catalog_1
      @@current_catalog = @@catalog_2
    else
      @@current_catalog = @@catalog_1
    end
    status 200
  end


  get '/env/:name' do
    ENV[params[:name]]
  end

  get '/env' do
    ENV.to_hash.to_s
  end

  run! if app_file == $0
end

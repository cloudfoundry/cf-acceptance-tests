ENV['RACK_ENV'] ||= 'development'

require 'rubygems'
require 'sinatra/base'
require 'json'
require 'pp'

ID = ((ENV["VCAP_APPLICATION"] && JSON.parse(ENV["VCAP_APPLICATION"])["instance_id"]) || SecureRandom.uuid).freeze

require 'bundler'
Bundler.require :default, ENV['RACK_ENV'].to_sym

$stdout.sync = true
$stderr.sync = true

class ServiceBroker < Sinatra::Base

  @@catalog_1 = {
    'services' => [
      {'name'=>'test',
       'id'=> '1234',
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
           'name'=>'5tb',
           'id'=>'2451fa22-df16-4c10-ba6e-1f682d3dcdc9', #randomize
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
      {'name'=>'test_2',
       'id'=> '1234',
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
           'name'=>'6tb',
           'id'=>'2451fa22-df16-4c10-ba6e-1f682d3dcdc9', #randomize
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

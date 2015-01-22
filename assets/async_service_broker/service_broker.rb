ENV['RACK_ENV'] ||= 'development'

require 'rubygems'
require 'sinatra/base'
require 'json'
require 'pp'
require 'logger'

$log = Logger.new('service_broker.log','weekly')

ID = ((ENV["VCAP_APPLICATION"] && JSON.parse(ENV["VCAP_APPLICATION"])["instance_id"]) || SecureRandom.uuid).freeze

require 'bundler'
Bundler.require :default, ENV['RACK_ENV'].to_sym

CONFIG_DATA = { 'service' => {}, 'plan' => {}, 'dashboard_client' => {} }

$stdout.sync = true
$stderr.sync = true

SERVICE_INSTANCE_PROGRESS = 0

class ServiceBroker < Sinatra::Base
  set :logging, true

  configure :production, :developmemt, :test do
    $log = Logger.new(STDOUT)

    begin
      CONFIG_DATA = JSON.parse(ENV['CONFIG'])
    rescue => e
      $log.info("Error loading config data as JSON")
    end

    CONFIG_DATA.merge!({ 'service' => {} }) unless CONFIG_DATA.has_key?('service')
    CONFIG_DATA.merge!({ 'plan' => {} }) unless CONFIG_DATA.has_key?('plan')
    CONFIG_DATA.merge!({ 'dashboard_client' => {} }) unless CONFIG_DATA.has_key?('dashboard_client')
  end

  configure :test do
    $log.level = Logger::INFO
    $log.info "log configured for test"
  end

  configure :production do
    $log.level = Logger::WARN
    $log.info "log configured for production"
  end

  configure :development do
    $log.level = Logger::DEBUG
    $log.info "log configured for development"
  end

  def dashboard_client
    {
      'id'           => 'sso-test',
      'secret'       => 'sso-secret',
      'redirect_uri' => 'http://localhost:5551'
    }.merge(CONFIG_DATA['dashboard_client'])
  end

  def log(request)
   $log.info "#{request.env['REQUEST_METHOD']} #{request.env['PATH_INFO']} #{request.env['QUERY_STRING']}"
  end

  def plans
    plan_template = {
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
    }

    CONFIG_DATA.fetch('plans', [plan_template]).map do |plan|
      plan_template.merge(plan)
    end
  end

  def catalog
    log(request)
    {
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
        'dashboard_client' => dashboard_client,
        'plan_updateable' => true,
        'plans' => plans,
      }.merge(CONFIG_DATA['service'])
    ]
  }
  end

  get '/v2/catalog/?' do
    log(request)
    catalog.to_json
  end

  put '/v2/service_instances/:id/?' do
    log(request)
    status 201
    SERVICE_INSTANCE_PROGRESS = 0
    {state: 'in progress'}.to_json
  end

  get '/v2/service_instances/:id/?' do
    log(request)
    status 201
    if SERVICE_INSTANCE_PROGRESS < 2
      response = {state: 'in progress', state_description: "#{SERVICE_INSTANCE_PROGRESS * 10}% done"}.to_json
      SERVICE_INSTANCE_PROGRESS += 1
      response
    else
      {state: 'succeeded', state_description: "100% done"}.to_json
    end
  end

  patch '/v2/service_instances/:id/?' do
    log(request)
    status 200
    {}.to_json
  end

  delete '/v2/service_instances/:id/?' do
    log(request)
    status 200
    {}.to_json
  end

  get '/env/:name' do
    log(request)
    ENV[params[:name]]
  end

  get '/env' do
    log(request)
    ENV.to_hash.to_s
  end

  put '/v2/service_instances/:instance_id/service_bindings/:id' do |instance_id, binding_id|
    log(request)
    content_type :json

    begin
      status 201
      {
          "credentials" => {
            "uri" => "fake-service://fake-user:fake-password@fake-host:3306/fake-dbname",
            "username" => "fake-user",
            "password" => "fake-password",
            "host" => "fake-host",
            "port" => 3306,
            "database" => "fake-dbname"
          }
      }.to_json
    rescue => e
      status 502
      {"description" => e.message}.to_json
    end
  end

  delete '/v2/service_instances/:instance_id/service_bindings/:id' do |instance_id, binding_id|
    log(request)
    content_type :json

    begin
      status 200
      {}.to_json
    rescue => e
      status 502
      {"description" => e.message}.to_json
    end
  end

  run! if app_file == $0
end

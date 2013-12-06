ENV['RACK_ENV'] = 'test'

$: << File.expand_path("../../.", __FILE__)

require 'dora'
require 'rspec'
require 'rack/test'

RSpec.configure do |conf|
  conf.include Rack::Test::Methods

  def app
    Dora
  end
end
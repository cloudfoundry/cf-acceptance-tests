require 'spec_helper'
require 'json'

describe ServiceBroker do

  describe "GET /v2/catalog" do
    it 'returns a non-empty catalog' do
      get '/v2/catalog'
      response = last_response
      expect(response.body).to be
      expect(JSON.parse(response.body)).to be
    end
  end

  describe "POST /v2/catalog" do
    it 'changes the catalog' do
      get '/v2/catalog'
      first_response = last_response
      expect(first_response.body).to be

      post '/v2/catalog'

      get '/v2/catalog'
      second_response = last_response
      expect(second_response.body).to_not eq(first_response.body)
    end
  end
end

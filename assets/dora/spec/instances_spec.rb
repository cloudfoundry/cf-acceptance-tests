require "spec_helper"

describe Instances do
  describe "GET /instances" do

  end

  describe "GET /id" do
    it "should get the instance id from the VCAP_APPLICATION json" do
      get "/id"
      expect(last_response.body).to eq "b4ffb1a7b677447296f9ff7eea535c43"
    end
  end
end
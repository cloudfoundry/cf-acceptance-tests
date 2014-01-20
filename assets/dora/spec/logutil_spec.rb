require "spec_helper"

describe LogUtils do
  describe "GET /loglines" do
    let(:id) { "b4ffb1a7b677447296f9ff7eea535c43" }

    it "should output one line" do
      get "/loglines/1"
      expect(last_response.body).to eq "logged 1 line to stdout"
    end
  end
end
require "spec_helper"

describe LogUtils do
  describe "GET /loglines" do
    it "should output one line" do
      get "/loglines/1"
      expect(last_response.body).to eq "logged 1 line to stdout"
    end

    it "should annotate lines" do
      get "/loglines/1/unique_tag"
      expect(last_response.body).to eq "logged 1 line with tag unique_tag to stdout"
    end
  end
end
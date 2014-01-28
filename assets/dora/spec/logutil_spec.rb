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

  describe "GET /log/sleep/count" do
    it "should return 0 if no loglines were created" do
      get "/log/sleep/count"
      expect(last_response.body).to eq "0"
    end

    it "should return the number of loglines created" do
      get "/log/sleep/100/limit/5"
      get "/log/sleep/count"
      expect(last_response.body.to_i).to eq 5
    end
  end  
end
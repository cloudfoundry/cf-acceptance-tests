require "spec_helper"

describe Dora do
  describe "GET /ready" do
    context "when readiness is already true" do
      before do
        get "/ready/true"
      end

      it "should be ready" do
        get "/ready"
        expect(last_response.body).to eq "200 - ready"
      end

      it "should set readiness to false" do
        get "/ready/false"
        get "/ready"
        expect(last_response.body).to eq "500 - not ready"
      end
    end

    context "when readiness is already false" do
      before do
        get "/ready/false"
      end

      it "should not be ready" do
        get "/ready"
        expect(last_response.body).to eq "500 - not ready"
        expect(last_response.status).to eq 500
      end

      it "should set readiness to true" do
        get "/ready/true"
        get "/ready"
        expect(last_response.body).to eq "200 - ready"
      end

      it "should set readiness to true with weird values" do
        get "/ready/meowpotatoblargasdf"
        get "/ready"
        expect(last_response.body).to eq "200 - ready"
      end
    end
  end
end

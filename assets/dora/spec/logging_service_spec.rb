require "spec_helper"

describe LoggingService do
  describe "produce_logspeed_output" do
    it "should write log message to the output" do
      logging_service = LoggingService.new
      fakestring = StringIO.new
      logging_service.output = fakestring
      logging_service.produce_logspeed_output(1,1,"foo")
      fakestring.rewind
      fakestring.lines.each {|l| puts l }
      expect(fakestring.lines).to eq 10

    end
  end
end
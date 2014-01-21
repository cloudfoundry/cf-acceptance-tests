class LogUtils < Sinatra::Base

  get "/loglines/:linecount" do
    produce_log_output(params[:linecount])
    "logged #{params[:linecount]} line to stdout"
  end

  get "/loglines/:linecount/:tag" do
    produce_log_output(params[:linecount], params[:tag])
    "logged #{params[:linecount]} line with tag #{params[:tag]} to stdout"
  end

  private
  def produce_log_output(linecount, tag="")
    linecount.to_i.times do |i|
      puts "#{Time.now.strftime("%FT%T.%N%:z")} line #{i} #{tag}"
      $stdout.flush
    end
  end
end

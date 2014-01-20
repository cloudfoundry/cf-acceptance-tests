class LogUtils < Sinatra::Base
  get "/loglines/:linecount" do
    params[:linecount].to_i.times do |i|
      puts "#{Time.now.strftime("%FT%T.%N%:z")} line #{i}"
    end
    "logged #{params[:linecount]} line to stdout"
  end

end

import sys
from BaseHTTPServer import BaseHTTPRequestHandler,HTTPServer

port = int(sys.argv[1])

class requestHandler(BaseHTTPRequestHandler):
    def do_GET(s):
        s.send_response(200)
        s.send_header("Content-type", "text/html")
        s.end_headers()
        s.wfile.write("<html>")
        s.wfile.write("<head><title>Python app.</title></head>")
        s.wfile.write("<body><p>python, world</p></body>")
        s.wfile.write("</html>")
        return

web_server = HTTPServer(('0.0.0.0', port), requestHandler)
print 'Started on port', port
web_server.serve_forever()

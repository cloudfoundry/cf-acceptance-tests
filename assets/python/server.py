import sys

if int(sys.version[0]) >= 3:
    from http.server import BaseHTTPRequestHandler, HTTPServer
else:
    from BaseHTTPServer import BaseHTTPRequestHandler,HTTPServer

port = int(sys.argv[1])

class requestHandler(BaseHTTPRequestHandler):
    def do_GET(s):
        s.send_response(200)
        s.send_header("Content-type", "text/html")
        s.end_headers()
        s.wfile.write("<html>".encode("utf-8"))
        s.wfile.write("<head><title>Python app.</title></head>".encode("utf-8"))
        s.wfile.write("<body><p>Hello, World</p></body>".encode("utf-8"))
        s.wfile.write("</html>".encode("utf-8"))
        return

web_server = HTTPServer(('0.0.0.0', port), requestHandler)
print('Started on port', port)
web_server.serve_forever()

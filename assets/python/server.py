from http.server import HTTPServer, BaseHTTPRequestHandler
from socketserver import ThreadingMixIn
import os

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.end_headers()
        message =  "Hello python, world!"
        self.wfile.write(message.encode('utf-8'))
        self.wfile.write('\n'.encode('utf-8'))
        return

class ThreadedHTTPServer(ThreadingMixIn, HTTPServer):
    """Handle requests in a separate thread."""

if __name__ == '__main__':
    port = int(os.environ.get('PORT', '8080'))
    host = os.environ.get('VCAP_APP_HOST', '127.0.0.1')
    print("Going to start server on %s:%s" % (host, port))
    server = ThreadedHTTPServer((host, port), Handler)
    print('Starting server, use <Ctrl-C> to stop')
    server.serve_forever()

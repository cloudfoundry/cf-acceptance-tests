from BaseHTTPServer import HTTPServer, BaseHTTPRequestHandler
from SocketServer import ThreadingMixIn
import os

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.end_headers()
        message =  "Hello python, world!"
        self.wfile.write(message)
        self.wfile.write('\n')
        return

class ThreadedHTTPServer(ThreadingMixIn, HTTPServer):
    """Handle requests in a separate thread."""

if __name__ == '__main__':
    port = int(os.environ.get('PORT', '8080'))
    host = os.environ.get('VCAP_APP_HOST', '127.0.0.1')
    print "Goin to start sever on %s:%s" % (host, port)
    server = ThreadedHTTPServer((host, port), Handler)
    print 'Starting server, use <Ctrl-C> to stop'
    server.serve_forever()

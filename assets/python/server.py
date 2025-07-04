import os
import logging
import http.client
from http.server import HTTPServer, BaseHTTPRequestHandler
from socketserver import ThreadingMixIn

DEFAULT_PORT = '8080'
HOST = '127.0.0.1'

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s | %(levelname)s | %(message)s'
)

def fetch_ip(endpoint):
    try:
        connection = http.client.HTTPConnection(endpoint, timeout=0.20)
        connection.request("GET", "/")
        response = connection.getresponse()

        if response.status == 200:
            ip_data = response.read().strip().decode('utf-8')
            connection.close()
            return 200, ip_data
        else:
            connection.close()
            return 500, f"Error: received status code {response.status}"

    except Exception as e:
        logging.error(f"Failed to fetch IP from {endpoint}: {e}")
        return 500, f"Error fetching IP: {e}"

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        endpoints = {
            '/ipv4-test': 'api.ipify.org',
            '/ipv6-test': 'api6.ipify.org',
            '/dual-stack-test': 'api64.ipify.org'
        }

        endpoint = endpoints.get(self.path)
        if endpoint:
            status, message = fetch_ip(endpoint)
            self.send_response(status)
            self.end_headers()
            self.wfile.write(message.encode('utf-8'))
        else:
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b'Hello python, world!\n')

class ThreadedHTTPServer(ThreadingMixIn, HTTPServer):
    """Handle requests in a separate thread."""

if __name__ == '__main__':
    port = int(os.environ.get('PORT', DEFAULT_PORT))
    host = os.environ.get('VCAP_APP_HOST', HOST)

    print(f"Going to start server on {host}:{port}")
    server = ThreadedHTTPServer((host, port), Handler)
    print('Starting server, use <Ctrl-C> to stop')

    server.serve_forever()
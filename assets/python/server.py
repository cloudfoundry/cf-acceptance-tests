import os
import logging
import ipaddress
import http.client

from http.server import HTTPServer, BaseHTTPRequestHandler
from socketserver import ThreadingMixIn

ENDPOINT_TYPE_MAP = {
    'api.ipify.org': {
        'validation_name': "IPv4",
        'path': "/ipv4-test"
    },
    'api6.ipify.org': {
        'validation_name': "IPv6",
        'path': "/ipv6-test"
    },
    'api64.ipify.org': {
        'validation_name': "Dual stack",
        'path': "/dual-stack-test"
    }
}

DEFAULT_PORT = '8080'
HOST = '127.0.0.1'

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s | %(levelname)s | %(message)s'
)

class IPv6Tester:
    """
    The `IPv6Tester` class is responsible for verifying the successful execution of
    egress calls using IPv4, IPv6, and Dual Stack configurations, depending on the input.
    It offers logging result from the call.
    The test execution is deemed successful if the requested endpoint is reached without errors.

    """

    def __init__(self, endpoints):
        self.endpoints = endpoints

    def test_single_address(self, endpoint):
        result = self.test_endpoint(endpoint)
        self.print_result(endpoint, result)
        return result

    def print_result(self, endpoint, result):
        validation_type = ENDPOINT_TYPE_MAP[endpoint]['validation_name']
        if result['success']:
            logging.info(f"{validation_type} validation succeeded.")
        else:
            logging.error(f"{validation_type} validation failed.")

    def test_endpoint(self, endpoint):
        try:
            logging.info(f"Testing endpoint: {endpoint}")
            connection = http.client.HTTPConnection(endpoint, timeout=0.20)
            connection.request("GET", "/")
            
            response = connection.getresponse()
            response_data = response.read().strip().decode('utf-8')
            ip_type = self.determine_ip_type(response_data)

            connection.close()
            return {
                'success': response.status == 200,
                'ip_type': ip_type
            }

        except Exception as e:
            logging.error(f"Failed to reach {endpoint}: {e}")
            return {
                'success': False,
                'error': str(e),
                'ip_type': 'Unknown'
            }

    @staticmethod
    def determine_ip_type(ip_string):
        try:
            ip = ipaddress.ip_address(ip_string)
            return "IPv4" if ip.version == 4 else "IPv6"
        except ValueError:
            return "Invalid IP"

class Handler(BaseHTTPRequestHandler):
    '''
        The Handler class provides two distinct test paths
        for different testing scenarios. The ipv6-path is dedicated
        to testing IPv6 egress calls, while the default path is used
        for testing the default Hello-Python buildpack test case.
    '''

    def do_GET(self):
        if self.path in [data['path'] for data in ENDPOINT_TYPE_MAP.values()]:
            self.handle_test()
        else:
            self.send_response(200)
            self.end_headers()
            message = "Hello python, world!"
            self.wfile.write(message.encode('utf-8'))
            self.wfile.write('\n'.encode('utf-8'))

    def handle_test(self):
        endpoint = self.get_endpoint_from_path()
        if endpoint:
            tester = IPv6Tester([endpoint])
            result = tester.test_single_address(endpoint)
            response_code = 200 if result['success'] else 500
            self.send_response(response_code)
            self.end_headers()

            validation_name = ENDPOINT_TYPE_MAP[endpoint]['validation_name']
            response_message = (f"{validation_name} validation resulted in "
                                f"{'success' if result['success'] else 'failure'}. Detected IP type is "
                                f"{result.get('ip_type', 'unknown')}. Error message: {result.get('error', 'none')}.")
            self.wfile.write(response_message.encode('utf-8'))
            self.wfile.write('\n'.encode('utf-8'))
        else:
            self.send_response(404)
            self.end_headers()
            self.wfile.write(b'Endpoint not found\n')

    def get_endpoint_from_path(self):
        for endpoint, data in ENDPOINT_TYPE_MAP.items():
            if self.path == data['path']:
                return endpoint
        return None

class ThreadedHTTPServer(ThreadingMixIn, HTTPServer):
    """Handle requests in a separate thread."""

if __name__ == '__main__':
    port = int(os.environ.get('PORT', DEFAULT_PORT))
    host = os.environ.get('VCAP_APP_HOST', HOST)

    print("Going to start server on %s:%s" % (host, port))
    server = ThreadedHTTPServer((host, port), Handler)
    print('Starting server, use <Ctrl-C> to stop')

    server.serve_forever()
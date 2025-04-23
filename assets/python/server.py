import os
import sys
import logging
import ipaddress
import http.client

from http.server import HTTPServer, BaseHTTPRequestHandler
from socketserver import ThreadingMixIn

ENDPOINT_TYPE_MAP = {
            'api.ipify.org': "IPv4",
            'api6.ipify.org': "IPv6",
            'api64.ipify.org': "Dual stack"
        }

PORT = '8080'
HOST = '127.0.0.1'

# Set up logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s | %(levelname)s | %(message)s'
)

class IPv6Tester:
    """
    The `IPv6Tester` class is responsible for verifying the successful execution of
    egress calls using IPv4, IPv6, and Dual Stack configurations, sequentially.
    It offers logging at each step to track the progress of the calls.
    The test execution is deemed successful if all endpoints are reached without errors.
    Conversely, if any egress call fails, the test execution is marked as failed,
    and the application exits with an exit code of 1 to signal the failure.
    """

    def __init__(self, endpoints):
        self.endpoints = endpoints
    
    def test_all_addresses(self):
        all_successful = True
        for endpoint in self.endpoints:
            result = self.test_endpoint(endpoint)
            self.print_result(endpoint, result)
            if not result['success']:
                all_successful = False

        if all_successful:
            logging.info("Test execution has completed. IPv6 validation is successful.")
        else:
            logging.error("Test execution has completed. IPv6 validation failed.")
            sys.exit(1)
    
    def print_result(self, endpoint, result):
        validation_type = ENDPOINT_TYPE_MAP.get(endpoint, "Unknown")
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
        path = self.path
        if path == "/ipv6-test":
            self.handle_ipv6_test()
        else:
            self.send_response(200)
            self.end_headers()
            message =  "Hello python, world!"
            self.wfile.write(message.encode('utf-8'))
            self.wfile.write('\n'.encode('utf-8'))
        
    def handle_ipv6_test(self):
        self.send_response(200)
        self.end_headers()
        tester = IPv6Tester(list(ENDPOINT_TYPE_MAP.keys()))
        tester.test_all_addresses()
        message = "IPv6 tests executed."
        self.wfile.write(message.encode('utf-8'))


class ThreadedHTTPServer(ThreadingMixIn, HTTPServer):
    """Handle requests in a separate thread."""


if __name__ == '__main__':
    port = int(os.environ.get('PORT', PORT))
    host = os.environ.get('VCAP_APP_HOST', HOST)

    print("Going to start server on %s:%s" % (host, port))
    server = ThreadedHTTPServer((host, port), Handler)
    print('Starting server, use <Ctrl-C> to stop')

    server.serve_forever()

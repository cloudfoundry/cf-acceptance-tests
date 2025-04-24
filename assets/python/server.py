import os
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

DEFAULT_PORT = '8080'
HOST = '127.0.0.1'

FAIL_MESSAGE = "Test execution has completed. IPv6 validation failed."
SUCCESS_MESSAGE = "Test execution has completed. IPv6 validation is successful."

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
        results = []
        all_successful = True
        for endpoint in self.endpoints:
            result = self.test_endpoint(endpoint)
            results.append((endpoint, result))
            self.print_result(endpoint, result)
            if not result['success']:
                all_successful = False

        if all_successful:
            logging.info(SUCCESS_MESSAGE)
        else:
            logging.error(FAIL_MESSAGE)
        
        return all_successful, results
    
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
        tester = IPv6Tester(list(ENDPOINT_TYPE_MAP.keys()))
        all_successful, results = tester.test_all_addresses()

        # Determine response status and message
        response_code = 200 if all_successful else 500
        overall_message = SUCCESS_MESSAGE if all_successful else FAIL_MESSAGE

        # Send HTTP response status
        self.send_response(response_code)
        self.end_headers()
       
        response_messages = []
        for endpoint, result in results:
            endpoint_results = f"{ENDPOINT_TYPE_MAP.get(endpoint, 'Unknown')} validation resulted in {'success' if result['success'] else 'failure'}. Detected IP type is {result.get('ip_type', 'unknown')}. Error message: {result.get('error', 'none')}."
            response_messages.append(endpoint_results)
        
        response_content = "\n".join(response_messages + [overall_message])
        
        # Write the detailed results and overall message to the web console
        self.wfile.write(response_content.encode('utf-8'))
        self.wfile.write('\n'.encode('utf-8'))
        


class ThreadedHTTPServer(ThreadingMixIn, HTTPServer):
    """Handle requests in a separate thread."""


if __name__ == '__main__':
    port = int(os.environ.get('PORT', DEFAULT_PORT))
    host = os.environ.get('VCAP_APP_HOST', HOST)

    print("Going to start server on %s:%s" % (host, port))
    server = ThreadedHTTPServer((host, port), Handler)
    print('Starting server, use <Ctrl-C> to stop')

    server.serve_forever()

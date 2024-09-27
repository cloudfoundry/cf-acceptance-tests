import os
import http.server
import socketserver
import sys

class CustomHandler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header('Content-type', 'text/html')
        self.end_headers()
        message = f"Hello, you've reached the instance {os.getenv('CF_INSTANCE_INDEX', 'not defined')}!"
        self.wfile.write(message.encode('utf-8'))

def main():
    port = int(os.getenv('PORT', 8080))
    instance_index = int(os.getenv('CF_INSTANCE_INDEX', -1))

    if instance_index > 1:
        print(f"Instance {instance_index} is quitting!")
        sys.exit(1)

    handler = CustomHandler
    httpd = socketserver.TCPServer(("", port), handler)

    print(f"Serving on port {port} from instance {instance_index}")
    httpd.serve_forever()

if __name__ == "__main__":
    main()

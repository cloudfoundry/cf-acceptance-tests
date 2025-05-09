var http = require('http');
var https = require('https');
var url = require('url');
var ip = require('ip');

HOST = null;

var host = "0.0.0.0";
var port = process.env.PORT || 3000;

const ENDPOINT_TYPE_MAP = {
    'api.ipify.org': {
        validation_name: "IPv4",
        path: "/ipv4-test"
    },
    'api6.ipify.org': {
        validation_name: "IPv6",
        path: "/ipv6-test"
    },
    'api64.ipify.org': {
        validation_name: "Dual stack",
        path: "/dual-stack-test"
    }
};

function testIPAddress(endpoint, expectedType) {
    return new Promise((resolve) => {
        https.get(`https://${endpoint}`, (resp) => {
            let data = '';

            resp.on('data', (chunk) => { data += chunk; });
            resp.on('end', () => {
                let success = false;
                let detectedType = 'unknown';

                if (expectedType === "IPv4" && ip.isV4Format(data)) {
                    success = true;
                    detectedType = "IPv4";
                } else if (expectedType === "IPv6" && ip.isV6Format(data)) {
                    success = true;
                    detectedType = "IPv6";
                } else if (expectedType === "Dual stack") {
                    if (ip.isV4Format(data)) {
                        success = true;
                        detectedType = "IPv4";
                    } else if (ip.isV6Format(data)) {
                        success = true;
                        detectedType = "IPv6";
                    }
                }

                resolve({
                    endpoint,
                    success,
                    ip_type: detectedType,
                    error: success ? 'none' : `Expected ${expectedType}, but got ${data}`
                });
            });
        }).on("error", (err) => {
            resolve({ endpoint, success: false, ip_type: 'unknown', error: err.message });
        });
    });
}

http.createServer(async function (req, res) {
    const parsedUrl = url.parse(req.url, true);
    const path = parsedUrl.pathname;

    let endpoint = null;
    for (const [ep, { path: epPath }] of Object.entries(ENDPOINT_TYPE_MAP)) {
        if (path === epPath) {
            endpoint = ep;
            break;
        }
    }

    if (endpoint) {
        const expectedType = ENDPOINT_TYPE_MAP[endpoint].validation_name;
        const result = await testIPAddress(endpoint, expectedType);

        const responseCode = result.success ? 200 : 500;
        res.writeHead(responseCode, { 'Content-Type': 'text/plain' });

        const responseMessage = `${expectedType} validation resulted in ${result.success ? 'success' : 'failure'}. Detected IP type is ${result.ip_type}. Error message: ${result.error}.`;

        res.end(responseMessage);

    } else {
        res.writeHead(200, { 'Content-Type': 'text/plain' });
        res.write('<h1>Hello from a node app! ');
        res.write('via: ' + host + ':' + port);
        res.end('</h1>');
    }

}).listen(port, host, () => {
    console.log('Server running at http://' + host + ':' + port + '/');
});

var http = require('http');
var https = require('https');

var host = "0.0.0.0";
var port = process.env.PORT || 3000;

http.createServer(async function (req, res) {
    const path = req.url;
    const endpoints = {
        '/ipv4-test': 'api.ipify.org',
        '/ipv6-test': 'api6.ipify.org',
        '/dual-stack-test': 'api64.ipify.org'
    };

    const endpoint = endpoints[path];

    if (endpoint) {
        https.get(`https://${endpoint}`, (resp) => {
            let data = '';

            resp.on('data', (chunk) => data += chunk);
            resp.on('end', () => {
                res.writeHead(200, { 'Content-Type': 'text/plain' });
                res.end(data.trim());
            });

        }).on("error", (err) => {
            res.writeHead(500, { 'Content-Type': 'text/plain' });
            res.end(err.message);
        });
    } else {
        res.writeHead(200, { 'Content-Type': 'text/plain' });
        res.end('Hello from a node app! ');
    }

}).listen(port, host, () => {
    console.log('Server running at http://' + host + ':' + port + '/');
});

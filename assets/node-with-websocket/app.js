var WebSocketServer = require('ws').Server
  , wss = new WebSocketServer({port: process.env.PORT || 3000});
wss.on('connection', function(ws) {
  ws.on('message', function(message) {
    console.log('received: %s', message);
    ws.send("Back");
  });
  ws.send('something');
});

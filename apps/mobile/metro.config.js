const { getDefaultConfig } = require('expo/metro-config');
const { createProxyMiddleware } = require('http-proxy-middleware');

const config = getDefaultConfig(__dirname);



console.log("\x1b[31mpour que cela fontionne,  il faut appeller \"http://10.208.113.46:8081\" dans le navigateur web et android Sinon, aura des erreur cors.\x1b[0m")

Atomics.wait(new Int32Array(new SharedArrayBuffer(4)), 0, 0, 5000);

config.server = {
  ...config.server,
  enhanceMiddleware: (middleware) => {
    return (req, res, next) => {
      console.log('Proxying request:', req.url);
      if (req.url.startsWith('/api')) {

        createProxyMiddleware({
          target: 'http://localhost:3300',
          changeOrigin: true,
          on: {
            proxyReq: (proxyReq, req) => {
              proxyReq.setHeader('X-Real-IP', req.socket.remoteAddress ?? '127.0.0.1');
            },
            proxyRes: (proxyRes) => {
              delete proxyRes.headers['access-control-allow-origin'];
              delete proxyRes.headers['access-control-allow-credentials'];
              delete proxyRes.headers['access-control-allow-methods'];
              delete proxyRes.headers['access-control-allow-headers'];
            },
          },
        })(req, res, next);
      } else {
        middleware(req, res, next);
      }
    };
  },
};

module.exports = config;

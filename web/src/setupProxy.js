const { createProxyMiddleware } = require('http-proxy-middleware');

// setup a proxy server for local development
module.exports = function(app) {
  app.use(
    '/knut.service.KnutService',
    createProxyMiddleware({
      target: 'http://localhost:7777/knut.service.KnutService',
      changeOrigin: true,
    })
  );
};
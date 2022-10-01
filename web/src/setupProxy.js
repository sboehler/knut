const { createProxyMiddleware } = require("http-proxy-middleware");

// setup a proxy server for local development
module.exports = function (app) {
  const mw = createProxyMiddleware({
    target: "http://127.0.0.1:7777",
    changeOrigin: true,
  });
  app.use("/knut.service.KnutService/", mw);
};

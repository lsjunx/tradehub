package config

import "flag"

// Config gateway 启动配置。
type Config struct {
	SaasURL            string // SaaS WSS 地址，如 wss://host:8443/ws/gateway
	Listen             string // Board TLS 监听，默认 :9443
	CertFile           string
	KeyFile            string
	DataDir            string // gateway_id 持久化目录
	Mode               string // development | production
	InsecureSkipVerify bool   // 仅 development 允许 true
}

func ParseFlags() Config {
	var c Config
	flag.StringVar(&c.SaasURL, "saas", "wss://127.0.0.1:8443/ws/gateway", "saas gateway websocket url")
	flag.StringVar(&c.Listen, "listen", ":9443", "board TLS listen address")
	flag.StringVar(&c.CertFile, "cert", "../certs/server.crt", "TLS cert")
	flag.StringVar(&c.KeyFile, "key", "../certs/server.key", "TLS key")
	flag.StringVar(&c.DataDir, "data-dir", "./data", "persistent data directory")
	flag.StringVar(&c.Mode, "mode", "development", "development|production")
	flag.BoolVar(&c.InsecureSkipVerify, "insecure-skip-verify", false, "skip TLS verify (dev only)")
	flag.Parse()
	return c
}

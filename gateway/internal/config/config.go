package config

import "flag"

// Config gateway 启动配置。
type Config struct {
	SaasURL            string
	Listen             string
	CertFile           string
	KeyFile            string
	DataDir            string
	Mode               string
	InsecureSkipVerify bool
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

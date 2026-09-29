package config

import "flag"

// Config saas-server 启动配置。
type Config struct {
	Addr     string
	CertFile string
	KeyFile  string
	Mode     string
}

func ParseFlags() Config {
	var c Config
	flag.StringVar(&c.Addr, "addr", ":8443", "HTTPS listen address")
	flag.StringVar(&c.CertFile, "cert", "../certs/server.crt", "TLS cert")
	flag.StringVar(&c.KeyFile, "key", "../certs/server.key", "TLS key")
	flag.StringVar(&c.Mode, "mode", "development", "development|production")
	flag.Parse()
	return c
}

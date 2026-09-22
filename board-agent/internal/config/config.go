package config

import (
	"flag"
	"time"
)

// Config board-agent 启动配置。
type Config struct {
	GatewayAddr       string
	DataDir           string
	ReportPort        int
	HeartbeatInterval time.Duration
	Mode              string
	InsecureSkipVerify bool
}

func ParseFlags() Config {
	var c Config
	flag.StringVar(&c.GatewayAddr, "gateway", "127.0.0.1:9443", "gateway host:port")
	flag.StringVar(&c.DataDir, "data-dir", "./data", "persistent data directory")
	flag.IntVar(&c.ReportPort, "report-port", 0, "port reported in REGISTER")
	flag.DurationVar(&c.HeartbeatInterval, "heartbeat", 10*time.Second, "heartbeat interval")
	flag.StringVar(&c.Mode, "mode", "development", "development|production")
	flag.BoolVar(&c.InsecureSkipVerify, "insecure-skip-verify", false, "skip TLS verify (dev only)")
	flag.Parse()
	return c
}

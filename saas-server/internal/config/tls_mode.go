package config

import (
	"fmt"
	"os"
)

// ValidateTLSMode 校验 mode 与 insecure_skip_verify 组合是否合法。
func ValidateTLSMode(mode string, insecureSkipVerify bool) error {
	switch mode {
	case "development":
		return nil
	case "production":
		if insecureSkipVerify {
			return fmt.Errorf("production mode forbids insecure_skip_verify=true")
		}
		return nil
	default:
		return fmt.Errorf("unknown mode %q (want development or production)", mode)
	}
}

// WarnIfInsecure 在关闭证书校验时打印固定 WARNING。
func WarnIfInsecure(insecureSkipVerify bool) {
	if insecureSkipVerify {
		fmt.Fprintln(os.Stderr, "WARNING: TLS certificate verification is disabled")
	}
}

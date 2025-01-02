//go:build !windows

package certpool

import (
	"crypto/x509"
)

func GetSystemCertPool() (*x509.CertPool, error) {
	return x509.SystemCertPool()
}

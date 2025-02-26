package servicetoolset

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sgostarter/i/commerr"
	"github.com/sgostarter/libeasygo/cuserror"
	"github.com/sgostarter/libeasygo/iputils"
	"github.com/sgostarter/libservicetoolset/certpool"
	"golang.org/x/net/http2"
)

type ClientAuthType int

// compat old logic

const (
	RequireAndVerifyClientCert ClientAuthType = iota
	NoClientCert
	RequestClientCert
	RequireAnyClientCert
	VerifyClientCertIfGiven
)

func ClientAuthTypeMap(ca ClientAuthType) tls.ClientAuthType {
	switch ca {
	case RequireAndVerifyClientCert:
		return tls.RequireAndVerifyClientCert
	case RequestClientCert:
		return tls.RequestClientCert
	case RequireAnyClientCert:
		return tls.RequireAnyClientCert
	case VerifyClientCertIfGiven:
		return tls.VerifyClientCertIfGiven
	case NoClientCert:
		fallthrough
	default:
		return tls.NoClientCert
	}
}

type GRPCServerTLSConfig struct {
	DisableSystemPool bool           `yaml:"DisableSystemPool" json:"disable_system_pool"`
	ClientAuth        ClientAuthType `yaml:"ClientAuth" json:"client_auth"`

	RootCAs [][]byte `yaml:"RootCAs" json:"root_cas" `
	Cert    []byte   `yaml:"Cert" json:"cert"`
	Key     []byte   `yaml:"Key" json:"key"`
}

type GRPCClientTLSConfig struct {
	DisableSystemPool  bool   `yaml:"DisableSystemPool" json:"disable_system_pool"`
	ServerName         string `yaml:"ServerName" json:"server_name"`
	InsecureSkipVerify bool   `yaml:"InsecureSkipVerify" json:"insecure_skip_verify"`

	RootCAs [][]byte `yaml:"RootCAs" json:"root_cas" `
	Cert    []byte   `yaml:"Cert" json:"cert"`
	Key     []byte   `yaml:"Key" json:"key"`
}

type GRPCServerTLSFileConfig struct {
	DisableSystemPool bool           `yaml:"DisableSystemPool" json:"disable_system_pool"`
	ClientAuth        ClientAuthType `yaml:"ClientAuth" json:"client_auth"`

	RootCAs []string `yaml:"RootCAs" json:"root_cas" `
	Cert    string   `yaml:"Cert" json:"cert"`
	Key     string   `yaml:"Key" json:"key"`
}

type GRPCClientTLSFileConfig struct {
	DisableSystemPool  bool   `yaml:"DisableSystemPool" json:"disable_system_pool"`
	ServerName         string `yaml:"ServerName" json:"server_name"`
	InsecureSkipVerify bool   `yaml:"InsecureSkipVerify" json:"insecure_skip_verify"`

	RootCAs []string `yaml:"RootCAs" json:"root_cas" `
	Cert    string   `yaml:"Cert" json:"cert"`
	Key     string   `yaml:"Key" json:"key"`
}

func GRPCServerTLSConfigMap(fileCfg *GRPCServerTLSFileConfig) (*GRPCServerTLSConfig, error) {
	if fileCfg == nil {
		return nil, nil
	}

	cfg := &GRPCServerTLSConfig{
		DisableSystemPool: fileCfg.DisableSystemPool,
		ClientAuth:        fileCfg.ClientAuth,
	}

	for _, ca := range fileCfg.RootCAs {
		d, err := os.ReadFile(ca)
		if err != nil {
			return nil, err
		}

		cfg.RootCAs = append(cfg.RootCAs, d)
	}

	if fileCfg.Cert != "" && fileCfg.Key != "" {
		d, err := os.ReadFile(fileCfg.Cert)
		if err != nil {
			return nil, err
		}

		cfg.Cert = d

		d, err = os.ReadFile(fileCfg.Key)
		if err != nil {
			return nil, err
		}

		cfg.Key = d
	}

	return cfg, nil
}

func GRPCClientTLSConfigMap(fileCfg *GRPCClientTLSFileConfig) (*GRPCClientTLSConfig, error) {
	if fileCfg == nil {
		return nil, nil
	}

	cfg := &GRPCClientTLSConfig{
		DisableSystemPool:  fileCfg.DisableSystemPool,
		ServerName:         fileCfg.ServerName,
		InsecureSkipVerify: fileCfg.InsecureSkipVerify,
	}

	for _, ca := range fileCfg.RootCAs {
		d, err := os.ReadFile(ca)
		if err != nil {
			return nil, err
		}

		cfg.RootCAs = append(cfg.RootCAs, d)
	}

	if fileCfg.Cert != "" && fileCfg.Key != "" {
		d, err := os.ReadFile(fileCfg.Cert)
		if err != nil {
			return nil, err
		}

		cfg.Cert = d

		d, err = os.ReadFile(fileCfg.Key)
		if err != nil {
			return nil, err
		}

		cfg.Key = d
	}

	return cfg, nil
}

func GenServerTLSConfig(cfg *GRPCServerTLSConfig) (tlsConfig *tls.Config, err error) {
	if cfg == nil {
		err = commerr.ErrInvalidArgument

		return
	}

	var caPool *x509.CertPool

	if cfg.DisableSystemPool {
		caPool = x509.NewCertPool()
	} else {
		caPool, err = certpool.GetSystemCertPool()
		if err != nil {
			return
		}
	}

	for _, ca := range cfg.RootCAs {
		caPool.AppendCertsFromPEM(ca)
	}

	cert, err := tls.X509KeyPair(cfg.Cert, cfg.Key)
	if err != nil {
		return
	}

	// nolint: gosec
	tlsConfig = &tls.Config{
		ClientAuth:   ClientAuthTypeMap(cfg.ClientAuth),
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caPool,
		NextProtos:   []string{http2.NextProtoTLS, "h2"},
	}

	return
}

func GenClientTLSConfig(cfg *GRPCClientTLSConfig) (tlsConfig *tls.Config, err error) {
	if cfg == nil {
		err = commerr.ErrInvalidArgument

		return
	}

	var caPool *x509.CertPool

	if cfg.DisableSystemPool {
		caPool = x509.NewCertPool()
	} else {
		caPool, err = certpool.GetSystemCertPool()
		if err != nil {
			return
		}
	}

	if len(cfg.RootCAs) > 0 {
		for _, ca := range cfg.RootCAs {
			caPool.AppendCertsFromPEM(ca)
		}
	}

	var clientCertificate tls.Certificate

	if len(cfg.Cert) > 0 && len(cfg.Key) > 0 {
		clientCertificate, err = tls.X509KeyPair(cfg.Cert, cfg.Key)
		if err != nil {
			return
		}
	}

	// nolint: gosec
	tlsConfig = &tls.Config{
		ServerName:         cfg.ServerName,
		Certificates:       []tls.Certificate{clientCertificate},
		RootCAs:            caPool,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
		NextProtos:         []string{http2.NextProtoTLS, "h2"},
	}

	return
}

func GetDiscoveryHostAndPort(externalAddress, listeningAddress string) (host string, port int, err error) {
	fnParse := func(address string) (host string, port int, err error) {
		if address == "" {
			err = commerr.ErrInvalidArgument

			return
		}

		vs := strings.Split(address, ":")
		if len(vs) > 2 {
			err = commerr.ErrInvalidArgument

			return
		}

		if len(vs) == 2 {
			host = vs[0]

			var port64 int64

			port64, err = strconv.ParseInt(vs[1], 10, 64)
			if err != nil {
				return
			}

			port = int(port64)
		} else {
			host = address
		}

		return
	}

	host, port, err = fnParse(externalAddress)
	if err != nil {
		host = ""
		port = 0
	}

	if host != "" && port > 0 {
		return
	}

	host2, port2, err := fnParse(listeningAddress)
	if err != nil {
		return
	}

	if host == "" {
		host = host2
	}

	if port <= 0 {
		port = port2
	}

	if host == "" {
		ips, errI := iputils.LocalIPv4s()
		if errI == nil && len(ips) > 0 {
			host = ips[0]
		}
	}

	if host == "" || port < 0 {
		err = cuserror.NewWithErrorMsg(fmt.Sprintf("invalid host port: %v,%v", host, port))

		return
	}

	return
}

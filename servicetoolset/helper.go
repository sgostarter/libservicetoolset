package servicetoolset

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"

	"github.com/sgostarter/libeasygo/commerr"
)

type GRPCTlsConfig struct {
	RootCAs    [][]byte `yaml:"RootCAs" json:"root_cas" `
	Cert       []byte   `yaml:"Cert" json:"cert"`
	Key        []byte   `yaml:"Key" json:"key"`
	ServerName string   `yaml:"ServerName" json:"server_name"`
}

type GRPCTlsFileConfig struct {
	RootCAs    []string `yaml:"RootCAs" json:"root_cas" `
	Cert       string   `yaml:"Cert" json:"cert"`
	Key        string   `yaml:"Key" json:"key"`
	ServerName string   `yaml:"ServerName" json:"server_name"`
}

func GRPCTlsConfigMap(fileCfg *GRPCTlsFileConfig) (*GRPCTlsConfig, error) {
	if fileCfg == nil {
		return nil, nil
	}

	if fileCfg.Cert == "" || fileCfg.Key == "" {
		return nil, commerr.ErrInvalidArgument
	}

	cfg := &GRPCTlsConfig{
		ServerName: fileCfg.ServerName,
	}

	for _, ca := range fileCfg.RootCAs {
		d, err := ioutil.ReadFile(ca)
		if err != nil {
			return nil, err
		}

		cfg.RootCAs = append(cfg.RootCAs, d)
	}

	d, err := ioutil.ReadFile(fileCfg.Cert)
	if err != nil {
		return nil, err
	}

	cfg.Cert = d

	d, err = ioutil.ReadFile(fileCfg.Key)
	if err != nil {
		return nil, err
	}

	cfg.Key = d

	return cfg, nil
}

func GenServerTLSConfig(cfg *GRPCTlsConfig) (tlsConfig *tls.Config, err error) {
	if cfg == nil {
		err = commerr.ErrInvalidArgument

		return
	}

	caPool := x509.NewCertPool()

	for _, ca := range cfg.RootCAs {
		caPool.AppendCertsFromPEM(ca)
	}

	cert, err := tls.X509KeyPair(cfg.Cert, cfg.Key)
	if err != nil {
		return
	}

	// nolint: gosec
	tlsConfig = &tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caPool,
	}

	return
}

func GenClientTLSConfig(cfg *GRPCTlsConfig) (tlsConfig *tls.Config, err error) {
	if cfg == nil {
		err = commerr.ErrInvalidArgument

		return
	}

	var caPool *x509.CertPool

	if len(cfg.RootCAs) > 0 {
		caPool = x509.NewCertPool()

		for _, ca := range cfg.RootCAs {
			caPool.AppendCertsFromPEM(ca)
		}
	}

	cert, err := tls.X509KeyPair(cfg.Cert, cfg.Key)
	if err != nil {
		return
	}

	// nolint: gosec
	tlsConfig = &tls.Config{
		ServerName:   cfg.ServerName,
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
	}

	return
}

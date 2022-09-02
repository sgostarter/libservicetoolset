package servicetoolset

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/sgostarter/libeasygo/commerr"
	"github.com/sgostarter/libeasygo/cuserror"
	"github.com/sgostarter/libeasygo/iputils"
)

type GRPCTlsConfig struct {
	RootCAs            [][]byte `yaml:"RootCAs" json:"root_cas" `
	Cert               []byte   `yaml:"Cert" json:"cert"`
	Key                []byte   `yaml:"Key" json:"key"`
	ServerName         string   `yaml:"ServerName" json:"server_name"`
	InsecureSkipVerify bool     `yaml:"InsecureSkipVerify" json:"insecure_skip_verify"`
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
		ClientAuth:         tls.RequireAndVerifyClientCert,
		Certificates:       []tls.Certificate{cert},
		ClientCAs:          caPool,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
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

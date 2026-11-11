package localcert

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	"github.com/trimble-oss/tierceron-hat/cap"
)

func readFirstExistingPEM(paths ...string) *[]byte {
	for _, path := range paths {
		if len(path) == 0 {
			continue
		}
		if pemBytes, err := os.ReadFile(path); err == nil && len(pemBytes) > 0 {
			return &pemBytes
		}
	}
	return nil
}

func nonEmptyPaths(paths ...string) []string {
	filtered := make([]string, 0, len(paths))
	for _, path := range paths {
		if len(path) > 0 {
			filtered = append(filtered, path)
		}
	}
	return filtered
}

func LoadFeatherTLSConfig(listenerCertPaths []string, listenerKeyPaths []string, rootCertPaths []string, serverName string) (*cap.FeatherTLSConfig, error) {
	listenerCertPaths = nonEmptyPaths(listenerCertPaths...)
	listenerKeyPaths = nonEmptyPaths(listenerKeyPaths...)
	rootCertPaths = nonEmptyPaths(rootCertPaths...)
	listenerCertPEM := readFirstExistingPEM(listenerCertPaths...)
	if listenerCertPEM == nil {
		return nil, fmt.Errorf("local feather certificate not found; looked in %s", strings.Join(listenerCertPaths, ", "))
	}
	listenerKeyPEM := readFirstExistingPEM(listenerKeyPaths...)
	if listenerKeyPEM == nil {
		return nil, fmt.Errorf("local feather key not found; looked in %s", strings.Join(listenerKeyPaths, ", "))
	}
	rootCertPEM := readFirstExistingPEM(rootCertPaths...)
	if rootCertPEM == nil {
		rootCertPEM = listenerCertPEM
	}
	if len(serverName) == 0 {
		return nil, fmt.Errorf("local feather TLS server name not provided")
	}

	listenerKeyPair, err := tls.X509KeyPair(*listenerCertPEM, *listenerKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("invalid local feather certificate/key pair: %w", err)
	}
	listenerCertificate, err := x509.ParseCertificate(listenerKeyPair.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("invalid local feather certificate: %w", err)
	}
	rootCertPool := x509.NewCertPool()
	if !rootCertPool.AppendCertsFromPEM(*rootCertPEM) {
		return nil, fmt.Errorf("local feather root certificate bundle contains no valid certificates")
	}
	intermediateCertPool := x509.NewCertPool()
	for _, certificateBytes := range listenerKeyPair.Certificate[1:] {
		intermediateCertificate, err := x509.ParseCertificate(certificateBytes)
		if err != nil {
			return nil, fmt.Errorf("invalid local feather intermediate certificate: %w", err)
		}
		intermediateCertPool.AddCert(intermediateCertificate)
	}
	if _, err := listenerCertificate.Verify(x509.VerifyOptions{
		DNSName:       serverName,
		Roots:         rootCertPool,
		Intermediates: intermediateCertPool,
	}); err != nil {
		return nil, fmt.Errorf("local feather certificate validation failed: %w", err)
	}

	tlsConfig := cap.NewFeatherPEMTLSConfig(listenerCertPEM, listenerKeyPEM, rootCertPEM)
	tlsConfig.ServerName = &serverName
	return tlsConfig, nil
}

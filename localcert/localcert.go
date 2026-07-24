package localcert

import (
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

	tlsConfig := cap.NewFeatherPEMTLSConfig(listenerCertPEM, listenerKeyPEM, rootCertPEM)
	tlsConfig.ServerName = &serverName
	return tlsConfig, nil
}

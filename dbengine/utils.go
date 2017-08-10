package dbengine

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net/http"
)

func makeDatabaseName(instanceID string) string {
	return fmt.Sprintf("db_%s", instanceID)
}

func makeUserName(bindingID string) string {
	return fmt.Sprintf("user_%s", bindingID)
}

func makeRandomPassword(desired_bytes_of_entropy int) (string, error) {
	buf := make([]byte, desired_bytes_of_entropy)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	// Convert to base64. The resulting password will therefore be longer than
	// `desired_bytes_of_entropy` but ASCII is safer to send between systems than
	// arbitrary binary data.
	return base64.URLEncoding.EncodeToString(buf), nil
}

func SetupHTTPClient(cert string) (*http.Client, error) {
	if cert == "" {
		return &http.Client{}, nil
	}

	roots := x509.NewCertPool()
	ca, err := base64.StdEncoding.DecodeString(cert)
	if err != nil {
		return nil, err
	}

	ok := roots.AppendCertsFromPEM(ca)
	if !ok {
		return nil, fmt.Errorf("root certificate: failed to parse")
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: roots,
			},
		},
	}

	return httpClient, nil
}

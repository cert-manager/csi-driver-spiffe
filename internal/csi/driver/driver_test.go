package driver

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math"
	"math/big"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
)

func Test_writeKeyFileWritesGoSPIFFECompatibleKeyFile(t *testing.T) {
	caUri, err := url.Parse("spiffe://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// create the CA cert
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	caSerial, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	caSubj := pkix.Name{
		Country:            []string{"GB"},
		Organization:       []string{"Example"},
		OrganizationalUnit: []string{"Engineering"},
		SerialNumber:       caSerial.String(),
	}
	caTemplate := &x509.Certificate{
		BasicConstraintsValid: true,
		SignatureAlgorithm:    x509.ECDSAWithSHA256,
		PublicKeyAlgorithm:    x509.ECDSA,
		PublicKey:             caKey.Public(),
		SerialNumber:          caSerial,
		Issuer:                caSubj,
		Subject:               caSubj,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(100 * time.Hour * 24 * 365),
		KeyUsage:              x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:                  true,
		DNSNames:              nil,
		EmailAddresses:        nil,
		IPAddresses:           nil,
		URIs:                  []*url.URL{caUri},
	}

	caCert, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, caKey.Public(), caKey)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	var tlsCert tls.Certificate

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	tlsCert.PrivateKey = leafKey

	leafSerial, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	leafSubj := pkix.Name{
		Country:            []string{"GB"},
		Organization:       []string{"Jetstack"},
		OrganizationalUnit: []string{"Product"},
		SerialNumber:       leafSerial.String(),
	}
	uri, err := url.Parse("spiffe://example.com/foo/bar")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	leafTemplate := &x509.Certificate{
		BasicConstraintsValid: true,
		SignatureAlgorithm:    x509.ECDSAWithSHA256,
		PublicKeyAlgorithm:    x509.ECDSA,
		PublicKey:             leafKey.Public(),
		SerialNumber:          leafSerial,
		Issuer:                caSubj,
		Subject:               leafSubj,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(99 * time.Hour * 24 * 365),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		IsCA:                  false,
		DNSNames:              nil,
		EmailAddresses:        nil,
		IPAddresses:           nil,
		URIs:                  []*url.URL{uri},
	}

	leafCert, err := x509.CreateCertificate(rand.Reader, leafTemplate, caTemplate, leafKey.Public(), caKey)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	tlsCert.Certificate = [][]byte{leafCert, caCert}

	// this is the function used by the driver
	keyPEM, err := pemFormatPrivateKey(tlsCert.PrivateKey)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	keyFilePath := "./key.pem"
	err = os.WriteFile(keyFilePath, keyPEM, 0644)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	defer os.Remove(keyFilePath)

	certPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: tlsCert.Certificate[0],
		},
	)
	certFilePath := "./cert.pem"
	err = os.WriteFile(certFilePath, certPEM, 0644)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	defer os.Remove(certFilePath)

	_, err = x509svid.Load(certFilePath, keyFilePath)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

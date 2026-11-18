package certificates

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

type Material struct {
	CertificatePEM []byte
	PrivateKeyPEM  []byte
	NotAfter       time.Time
}

func Generate(dnsNames []string, duration time.Duration) (*Material, error) {
	if len(dnsNames) == 0 {
		return nil, fmt.Errorf("dnsNames must not be empty")
	}
	if duration < time.Minute {
		return nil, fmt.Errorf("duration must be at least 1m")
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate private key: %w", err)
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return nil, fmt.Errorf("generate serial number: %w", err)
	}
	if serialNumber.Sign() == 0 {
		serialNumber.SetInt64(1)
	}

	now := time.Now().UTC().Truncate(time.Second)

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "mini-cert-operator",
			Organization: []string{"Mini Certificate Operator"},
		},
		DNSNames: append([]string(nil), dnsNames...),

		NotBefore: now.Add(-time.Minute),
		NotAfter:  now.Add(duration).Truncate(time.Second),

		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certificateDER, err := x509.CreateCertificate(
		rand.Reader,
		template,
		template,
		&privateKey.PublicKey,
		privateKey,
	)
	if err != nil {
		return nil, fmt.Errorf("create certificate: %w", err)
	}

	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("encode private key: %w", err)
	}

	return &Material{
		CertificatePEM: pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certificateDER,
		}),
		PrivateKeyPEM: pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: privateKeyDER,
		}),
		NotAfter: template.NotAfter,
	}, nil
}

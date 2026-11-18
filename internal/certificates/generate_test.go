package certificates

import (
	"crypto/tls"
	"crypto/x509"
	"testing"
	"time"
)

func TestGenerate(t *testing.T) {
	material, err := Generate(
		[]string{"demo.example.com"},
		24*time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}

	pair, err := tls.X509KeyPair(
		material.CertificatePEM,
		material.PrivateKeyPEM,
	)
	if err != nil {
		t.Fatalf("invalid certificate/key pair: %v", err)
	}

	certificate, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}

	if err := certificate.VerifyHostname("demo.example.com"); err != nil {
		t.Fatal(err)
	}

	if err := certificate.CheckSignature(
		certificate.SignatureAlgorithm,
		certificate.RawTBSCertificate,
		certificate.Signature,
	); err != nil {
		t.Fatal(err)
	}

	if !certificate.NotAfter.Equal(material.NotAfter) {
		t.Fatal("returned expiry differs from certificate expiry")
	}

	remaining := time.Until(certificate.NotAfter)
	if remaining < 24*time.Hour-time.Minute || remaining > 24*time.Hour {
		t.Fatalf("unexpected remaining lifetime: %s", remaining)
	}
}

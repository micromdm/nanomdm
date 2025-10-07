package test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"time"
)

func GenerateRandomCertificateSerialNumber() (*big.Int, error) {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, limit)
}

func SelfSignedCertRSAResigner(tmpl *x509.Certificate) (key *rsa.PrivateKey, cert *x509.Certificate, err error) {
	key, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return
	}
	cert, err = x509.ParseCertificate(certBytes)
	if err != nil {
		return
	}

	return

}

func SimpleSelfSignedRSAKeypair(cn string, days int) (key *rsa.PrivateKey, cert *x509.Certificate, err error) {
	serialNumber, err := GenerateRandomCertificateSerialNumber()
	if err != nil {
		return nil, nil, err
	}
	timeNow := time.Now()
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: cn,
		},
		NotBefore:             timeNow,
		NotAfter:              timeNow.Add(time.Duration(days) * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{cn},
	}

	return SelfSignedCertRSAResigner(&template)
}

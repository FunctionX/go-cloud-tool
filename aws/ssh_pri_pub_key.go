package aws

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"golang.org/x/crypto/ssh"
)

const DefBitSize = 2048

func GenSSHKey() (priKey, pubKey string, err error) {
	privateKey, err := GeneratePrivateKey(DefBitSize)
	if err != nil {
		return
	}

	publicKey, err := EncodePublicKey(&privateKey.PublicKey)
	if err != nil {
		return
	}

	privateKeyBytes := EncodePrivateKeyToPEM(privateKey)
	return string(privateKeyBytes), string(publicKey), nil
}

// GeneratePrivateKey creates a RSA Private Key of specified byte size
func GeneratePrivateKey(bitSize int) (*rsa.PrivateKey, error) {
	// Private Key generation
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, err
	}

	// Validate Private Key
	if err = privateKey.Validate(); err != nil {
		return nil, err
	}

	return privateKey, nil
}

// EncodePrivateKeyToPEM encodes Private Key from RSA to PEM format
func EncodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	// Get ASN.1 DER format
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)

	// pem.Block
	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	// Private key in PEM format
	privatePEM := pem.EncodeToMemory(&privBlock)

	return privatePEM
}

func EncodePublicKey(publicKey *rsa.PublicKey) ([]byte, error) {
	rsaPublicKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		return nil, err
	}

	return ssh.MarshalAuthorizedKey(rsaPublicKey), nil
}

package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"time"
)

type KeyPair struct {
	PrivKey  *rsa.PrivateKey
	DERBytes []byte
}

func NewDefaultCertificateSubject(cn string) pkix.Name {
	return pkix.Name{
		CommonName:         cn,
		Country:            []string{"US"},
		Province:           []string{"TX"},
		Locality:           []string{"Heart of"},
		Organization:       []string{"Acme Co Inc"},
		OrganizationalUnit: []string{"Cast-x"},
	}
}

func NewDefaultCertificateTemplate(subject pkix.Name) *x509.Certificate {
	notBefore := time.Now().AddDate(-1, 0, 0)
	notAfter := notBefore.AddDate(10, 0, 0)
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		panic(fmt.Errorf("Unable to create serial number: %v", err))
	}
	return &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               subject,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
}

// template is mutated
func GenerateRootCAKeyPair(template *x509.Certificate, privKey *rsa.PrivateKey) (kp *KeyPair, err error) {
	kp = &KeyPair{PrivKey: privKey}
	if template == nil {
		template = NewDefaultCertificateTemplate(NewDefaultCertificateSubject("Cast Root CA"))
	}
	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign
	if kp.PrivKey == nil {
		if kp.PrivKey, err = rsa.GenerateKey(rand.Reader, 2048); err != nil {
			return nil, fmt.Errorf("Unable to generate key: %v", err)
		}
	}
	template.SubjectKeyId = sha1Hash(kp.PrivKey.N)
	kp.DERBytes, err = x509.CreateCertificate(rand.Reader, template, template, &kp.PrivKey.PublicKey, kp.PrivKey)
	if err != nil {
		return nil, fmt.Errorf("Failed creating certificate: %v", err)
	}
	return
}

// template is mutated
func GenerateIntermediateCAKeyPair(
	parentCA *KeyPair,
	template *x509.Certificate,
	privKey *rsa.PrivateKey,
) (kp *KeyPair, err error) {
	kp = &KeyPair{PrivKey: privKey}
	if template == nil {
		template = NewDefaultCertificateTemplate(NewDefaultCertificateSubject("Cast Inter CA"))
	}
	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign
	if kp.PrivKey == nil {
		if kp.PrivKey, err = rsa.GenerateKey(rand.Reader, 2048); err != nil {
			return nil, fmt.Errorf("Unable to generate key: %v", err)
		}
	}
	template.SubjectKeyId = sha1Hash(kp.PrivKey.N)
	parentCACert, err := parentCA.CreateX509Certificate()
	if err != nil {
		return nil, fmt.Errorf("Failed parsing CA cert: %v", err)
	}
	kp.DERBytes, err = x509.CreateCertificate(rand.Reader, template, parentCACert,
		&kp.PrivKey.PublicKey, parentCA.PrivKey)
	if err != nil {
		return nil, fmt.Errorf("Failed creating certificate: %v", err)
	}
	return
}

// template is mutated
func GenerateStandardKeyPair(
	parentCA *KeyPair,
	template *x509.Certificate,
	privKey *rsa.PrivateKey,
) (kp *KeyPair, err error) {
	kp = &KeyPair{PrivKey: privKey}
	if template == nil {
		template = NewDefaultCertificateTemplate(NewDefaultCertificateSubject("Cast Cert"))
	}
	if kp.PrivKey == nil {
		if kp.PrivKey, err = rsa.GenerateKey(rand.Reader, 2048); err != nil {
			return nil, fmt.Errorf("Unable to generate key: %v", err)
		}
	}
	template.SubjectKeyId = sha1Hash(kp.PrivKey.N)
	parentCACert, err := parentCA.CreateX509Certificate()
	if err != nil {
		return nil, fmt.Errorf("Failed parsing CA cert: %v", err)
	}
	kp.DERBytes, err = x509.CreateCertificate(rand.Reader, template, parentCACert,
		&kp.PrivKey.PublicKey, parentCA.PrivKey)
	if err != nil {
		return nil, fmt.Errorf("Failed creating certificate: %v", err)
	}
	return
}

func (k *KeyPair) CreateX509Certificate() (*x509.Certificate, error) {
	return x509.ParseCertificate(k.DERBytes)
}

func (k *KeyPair) EncodeCertPEM() []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: k.DERBytes})
}

func (k *KeyPair) EncodeKeyPEM() []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k.PrivKey)})
}

func (k *KeyPair) PersistToFiles(certFile string, keyFile string) error {
	if err := ioutil.WriteFile(certFile, k.EncodeCertPEM(), 0600); err != nil {
		return err
	}
	return ioutil.WriteFile(keyFile, k.EncodeKeyPEM(), 0600)
}

func (k *KeyPair) CreateTLSCertificate() (tls.Certificate, error) {
	return tls.X509KeyPair(k.EncodeCertPEM(), k.EncodeKeyPEM())
}

func sha1Hash(n *big.Int) []byte {
	h := sha1.New()
	h.Write(n.Bytes())
	return h.Sum(nil)
}

package chrome

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"

	"github.com/cretz/owncast/owncast/cert"
	"github.com/cretz/owncast/owncast/log"
)

func GenerateReplacementRootCA(
	existingDERBytesLen int,
	template *x509.Certificate,
	privKey *rsa.PrivateKey,
) (*cert.KeyPair, error) {
	if template == nil {
		template = cert.NewDefaultCertificateTemplate(cert.NewDefaultCertificateSubject("Cast Root CA"))
	}
	if len(template.Subject.OrganizationalUnit) == 0 {
		template.Subject.OrganizationalUnit = []string{}
	}
	origOU := template.Subject.OrganizationalUnit[0]
	privGivenInParam := privKey != nil
	// Try X times to reach the size
	const maxTries = 10
	var err error
	for tries := 1; tries <= maxTries; tries++ {
		log.Debugf("Attempt %v/%v to generate key of %v bytes", tries, maxTries, existingDERBytesLen)
		// New priv key each try if not given
		if !privGivenInParam {
			if privKey, err = rsa.GenerateKey(rand.Reader, 2048); err != nil {
				return nil, fmt.Errorf("Unable to generate key: %v", err)
			}
		}
		// Each try just appends 'x' to the OU until we hit at least the size
		template.Subject.OrganizationalUnit[0] = origOU
		myDERBytesLen := 0
		for myDERBytesLen < existingDERBytesLen {
			kp, err := cert.GenerateRootCAKeyPair(template, privKey)
			if err != nil {
				return nil, err
			} else if myDERBytesLen == 0 && len(kp.DERBytes) > existingDERBytesLen {
				return nil, fmt.Errorf("Generated key size greater than existing on first try")
			}
			myDERBytesLen = len(kp.DERBytes)
			if myDERBytesLen == existingDERBytesLen {
				return kp, nil
			}
			template.Subject.OrganizationalUnit[0] += "x"
			log.Debugf("Key of size %v isn't %v, changed OU to %v",
				myDERBytesLen, existingDERBytesLen, template.Subject.OrganizationalUnit[0])
		}
	}
	return nil, fmt.Errorf("Tried %v times to reach size, failed", maxTries)
}

package chrome

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"

	"github.com/cretz/owncast/cert"
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
	// Try 50 times to reach the size
	for tries := 0; tries < 50; tries++ {
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
		}
	}
	return nil, fmt.Errorf("Tried 50 times to reach size, couldn't")
}

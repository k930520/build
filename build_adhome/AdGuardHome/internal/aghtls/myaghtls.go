package aghtls

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/netutil"
	"go.step.sm/crypto/keyutil"
	"go.step.sm/crypto/x509util"
	"golang.org/x/net/publicsuffix"
)

type CAPair struct {
	rootCert *x509.Certificate
	rootKey  crypto.PrivateKey
}

func (mgr *DefaultManager) myOnGetCertificate(
	chi *tls.ClientHelloInfo) (cert *tls.Certificate, err error,
) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	if !mgr.extTLSConf.Enabled || mgr.tlsConf == nil {
		return nil, nil
	}

	if mgr.rootCert.IsCA {
		serverName := chi.ServerName
		if serverName == "" {
			serverName = mgr.extTLSConf.ServerName
		}
		certificates := []*x509.Certificate{mgr.tlsCert.Leaf}
		for _, der := range mgr.tlsCert.Certificate[1:] {
			certificate, err := x509.ParseCertificate(der)
			if err != nil {
				return nil, err
			}
			certificates = append(certificates, certificate)
		}
		if validateCertChain(context.Background(), mgr.logger, mgr.RootCAs(), certificates, serverName) != nil {
			var sans []string
			for _, address := range mgr.tlsCert.Leaf.IPAddresses {
				sans = append(sans, address.String())
			}
			sans = append(sans, mgr.tlsCert.Leaf.DNSNames...)
			if !netutil.IsValidIPString(serverName) {
				tldPlusOne, err := publicsuffix.EffectiveTLDPlusOne(serverName)
				if err != nil {
					return nil, err
				}
				if tldPlusOne != serverName {
					serverName = "*." + tldPlusOne
				}
			}
			sans = append(sans, serverName)
			err := mgr.generateServerCert(sans)
			if err != nil {
				return nil, err
			}
		}
	}

	return mgr.tlsCert, nil
}

func (mgr *DefaultManager) generateServerCert(sans []string) error {

	template, leafKey, err := newCert(mgr.extTLSConf.ServerName, x509util.DefaultLeafTemplate, sans, 24*time.Hour)
	if err != nil {
		return err
	}
	leafCert, err := x509util.CreateCertificate(template, mgr.rootCert, leafKey.Public(), mgr.rootKey.(crypto.Signer))
	if err != nil {
		return err
	}

	mgr.tlsCert = &tls.Certificate{
		Certificate: [][]byte{leafCert.Raw},
		PrivateKey:  leafKey,
		Leaf:        leafCert,
	}

	return nil
}

func newCert(commonName, templateName string, sans []string, lifetime time.Duration) (cert *x509.Certificate, signer crypto.Signer, err error) {
	signer, err = keyutil.GenerateDefaultSigner()
	if err != nil {
		return nil, nil, err
	}
	csr, err := x509util.CreateCertificateRequest(commonName, sans, signer)
	if err != nil {
		return nil, nil, err
	}
	template, err := x509util.NewCertificate(csr, x509util.WithTemplate(templateName, x509util.CreateTemplateData(commonName, sans)))
	if err != nil {
		return nil, nil, err
	}
	cert = template.GetCertificate()
	cert.NotBefore = time.Now().Truncate(time.Second)
	cert.NotAfter = cert.NotBefore.Add(lifetime)
	return cert, signer, nil
}

func myValidateCertificates(
	ctx context.Context,
	logger *slog.Logger,
	tlsManager Manager,
	status *TLSConfigStatus,
	certChain []byte,
	pkey []byte,
	serverName string,
) (err error) {
	// Check only the public certificate separately from the key.
	if len(certChain) > 0 {
		var ok bool
		ok, err = myValidateCertificate(
			ctx,
			logger,
			tlsManager,
			status,
			certChain,
			serverName,
		)
		if !ok {
			// Don't wrap the error, since it's informative enough as is.
			return err
		}
	}

	// Validate the private key by parsing it.
	if len(pkey) > 0 {
		var keyErr error
		status.KeyType, keyErr = myValidatePKey(tlsManager, pkey)
		if keyErr != nil {
			// Don't wrap the error, since it's informative enough as is.
			return keyErr
		}

		// Set status.ValidKey to true to signal the frontend that the
		// key is valid.
		status.ValidKey = true
	}

	// If both are set, validate together.
	if len(certChain) > 0 && len(pkey) > 0 {
		_, pairErr := tls.X509KeyPair(certChain, pkey)
		if pairErr != nil {
			return fmt.Errorf("certificate-key pair: %w", pairErr)
		}

		status.ValidPair = true
	}

	return err
}

func myValidateCertificate(
	ctx context.Context,
	logger *slog.Logger,
	tlsManager Manager,
	status *TLSConfigStatus,
	certChain []byte,
	serverName string,
) (ok bool, err error) {
	// parseErr is a non-critical parse warning.
	var parseErr error
	var certs []*x509.Certificate

	// Set status.ValidCert to true to signal the frontend that the
	// certificate opens successfully and certificate chain is valid.
	certs, status.ValidCert, parseErr = parseCertChain(ctx, logger, certChain)
	if !status.ValidCert {
		// Don't wrap the error, since it's informative enough as is.
		return false, parseErr
	}

	mainCert := certs[0]
	status.Subject = mainCert.Subject.String()
	status.Issuer = mainCert.Issuer.String()
	status.NotAfter = mainCert.NotAfter
	status.NotBefore = mainCert.NotBefore
	status.DNSNames = mainCert.DNSNames

	if mainCert.IsCA {
		status.ValidChain = true
		tlsManager.(*DefaultManager).rootCert = mainCert
		return true, nil
	}

	err = validateCertChain(ctx, logger, tlsManager.RootCAs(), certs, serverName)
	if err != nil {
		// Let self-signed certs through and don't return this error to set
		// its message into the status.WarningValidation afterwards.
		return true, err
	}

	status.ValidChain = true

	// Propagate the non-critical parse warning.
	return true, parseErr
}

func myValidatePKey(tlsManager Manager, pkey []byte) (keyType string, err error) {
	var key *pem.Block

	// Go through all pem blocks, but take first valid pem block and drop the
	// rest.
	for decoded, pemblock := pem.Decode([]byte(pkey)); decoded != nil; {
		if decoded.Type == "PRIVATE KEY" || strings.HasSuffix(decoded.Type, " PRIVATE KEY") {
			key = decoded

			break
		}

		decoded, pemblock = pem.Decode(pemblock)
	}

	if key == nil {
		return "", errors.Error("no valid keys were found")
	}

	rootKey, keyType, err := parsePrivateKey(key.Bytes)
	if err != nil {
		return "", fmt.Errorf("parsing private key: %w", err)
	}

	if keyType == keyTypeED25519 {
		return "", errors.Error(
			"ED25519 keys are not supported by browsers; " +
				"did you mean to use X25519 for key exchange?",
		)
	}

	if tlsManager.(*DefaultManager).rootCert.IsCA {
		tlsManager.(*DefaultManager).rootKey = rootKey
	}

	return keyType, nil
}

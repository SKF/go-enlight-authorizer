package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/SKF/go-enlight-authorizer/client/credentialsmanager"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const CertificateGracePeriod = 24 * time.Hour

type autoRefreshingTransportCredentials struct {
	credentials           credentials.TransportCredentials
	cf                    credentialsmanager.CredentialsFetcher
	secretKeyName         string
	serverName            string
	certificateExpiryTime time.Time
}

func getCredentialOption(ctx context.Context, cf credentialsmanager.CredentialsFetcher, host, secretKeyName string) (grpc.DialOption, error) {
	c, err := NewAutoRefreshingTransportCredentials(ctx, cf, secretKeyName, host)
	if err != nil {
		return nil, err
	}

	return grpc.WithTransportCredentials(c), nil
}

func NewAutoRefreshingTransportCredentials(ctx context.Context, cf credentialsmanager.CredentialsFetcher, secretKeyName, host string) (credentials.TransportCredentials, error) {
	creds := &autoRefreshingTransportCredentials{
		secretKeyName: secretKeyName,
		cf:            cf,
		serverName:    host,
	}

	if err := creds.loadCertificates(ctx); err != nil {
		return nil, err
	}

	return creds, nil
}

func (c *autoRefreshingTransportCredentials) ensureValidCredentials(ctx context.Context) error {
	if !c.shouldLoadNewCertificates() {
		return nil
	}

	return c.loadCertificates(ctx)
}

func (c *autoRefreshingTransportCredentials) shouldLoadNewCertificates() bool {
	earliestReload := c.certificateExpiryTime.Add(-CertificateGracePeriod)

	return time.Now().After(earliestReload)
}

func (c *autoRefreshingTransportCredentials) loadCertificates(ctx context.Context) error {
	config, err := c.loadCertificateIntoConfig(ctx)
	if err != nil {
		return err
	}

	var expiryTime time.Time
	for _, chain := range config.Certificates {
		for _, certificate := range chain.Certificate {
			var parsedCertificate *x509.Certificate
			if parsedCertificate, err = x509.ParseCertificate(certificate); err != nil {
				return err
			}

			t := parsedCertificate.NotAfter
			if expiryTime == (time.Time{}) || t.Before(expiryTime) {
				expiryTime = t
			}
		}
	}

	c.credentials = credentials.NewTLS(config)
	c.certificateExpiryTime = expiryTime

	return nil
}

func (c *autoRefreshingTransportCredentials) loadCertificateIntoConfig(ctx context.Context) (*tls.Config, error) {
	secrets, err := c.cf.GetDataStore(ctx, c.secretKeyName)
	if err != nil {
		return nil, err
	}

	certificate, err := tls.X509KeyPair(secrets.Crt, secrets.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certs: %w", err)
	}

	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM(secrets.CA)
	if !ok {
		return nil, errors.New("failed to append certs")
	}

	return &tls.Config{
		ServerName:   c.serverName,
		Certificates: []tls.Certificate{certificate},
		RootCAs:      certPool,
	}, nil
}

func (c *autoRefreshingTransportCredentials) ClientHandshake(ctx context.Context, s string, conn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	if err := c.ensureValidCredentials(ctx); err != nil {
		return nil, nil, err
	}

	return c.credentials.ClientHandshake(ctx, s, conn)
}

func (c *autoRefreshingTransportCredentials) ServerHandshake(conn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	if err := c.ensureValidCredentials(context.Background()); err != nil {
		return nil, nil, err
	}

	return c.credentials.ServerHandshake(conn)
}

func (c *autoRefreshingTransportCredentials) Info() credentials.ProtocolInfo {
	if c.credentials == nil {
		return credentials.ProtocolInfo{}
	}

	return c.credentials.Info()
}

func (c *autoRefreshingTransportCredentials) Clone() credentials.TransportCredentials {
	return &autoRefreshingTransportCredentials{
		credentials:           c.credentials.Clone(),
		cf:                    c.cf,
		secretKeyName:         c.secretKeyName,
		serverName:            c.serverName,
		certificateExpiryTime: c.certificateExpiryTime,
	}
}

func (c *autoRefreshingTransportCredentials) OverrideServerName(s string) error {
	if c.credentials == nil {
		return nil
	}

	return c.credentials.OverrideServerName(s) //nolint:staticcheck
}

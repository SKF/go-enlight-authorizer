package client_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	_ "embed"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/SKF/go-enlight-authorizer/client"
	"github.com/SKF/go-enlight-authorizer/client/credentialsmanager"

	authorizeproto "github.com/SKF/proto/v2/authorize"
	"github.com/SKF/proto/v2/common"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

var (
	//go:embed certs/rsa-key.pem
	rsaKey []byte
)

const DistantFuture = 10 * 365 * 24 * time.Hour

func setup(t *testing.T, authorizeServer dummyAuthorizeServer) (credentialsmanager.DataStore, server) {
	privateKey, err := parseRSAKey()
	require.NoError(t, err)

	caCertPEM, err := generateCA(privateKey)
	require.NoError(t, err)

	serverDataStore, err := generateDatastore(ca, privateKey, caCertPEM, DistantFuture)
	require.NoError(t, err)

	clientDataStore, err := generateDatastore(ca, privateKey, caCertPEM, DistantFuture)
	require.NoError(t, err)

	tlsCredentials, err := loadTLSCredentials(serverDataStore)
	require.NoError(t, err)

	srv := newServer(tlsCredentials, authorizeServer)
	err = srv.Start()
	require.NoError(t, err)

	return clientDataStore, srv
}

func TestDeadline(t *testing.T) {
	clientDataStore, srv := setup(t, dummyAuthorizeServer{})

	c := client.CreateClient()
	c.SetRequestTimeout(0)

	childCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := c.DialUsingCredentialsManager(childCtx, &mockCredentialsFetcher{ds: clientDataStore}, "localhost", "10000", "")
	require.NoError(t, err)

	srv.Shutdown()

	// Perform a call without specifying any deadline. As the server has been shut down this
	// would normally result in the call blocking forever waiting for the client to reconnect.
	// Assert: The call is intercepted and the request timeout is injected as a deadline.

	_, err = c.GetResource(context.Background(), "", "")

	require.EqualError(t, err, "rpc error: code = DeadlineExceeded desc = context deadline exceeded",
		"Caller omits deadline and the default request timeout is used")
}

func TestReconnect(t *testing.T) {
	clientDataStore, srv := setup(t, dummyAuthorizeServer{})

	c := client.CreateClient()

	err := c.DialUsingCredentialsManager(context.Background(), &mockCredentialsFetcher{ds: clientDataStore}, "localhost", "10000", "")
	require.NoError(t, err)

	err = srv.Restart()
	require.NoError(t, err)
	defer srv.Shutdown()

	_, err = c.GetResource(context.Background(), "", "")

	require.NoError(t, err)
}

func TestRetryPolicy(t *testing.T) {
	clientDataStore, srv := setup(t, dummyAuthorizeServer{
		failuresRemaining: 4,
	})
	defer srv.Shutdown()

	c := client.CreateClient()

	err := c.DialUsingCredentialsManager(context.Background(), &mockCredentialsFetcher{ds: clientDataStore}, "localhost", "10000", "")
	require.NoError(t, err)

	_, err = c.GetResource(context.Background(), "", "")
	require.NoError(t, err)
}

func TestClientHandshake_CertificateAboutToExpire(t *testing.T) {
	privateKey, err := parseRSAKey()
	require.NoError(t, err)

	caCertPEM, err := generateCA(privateKey)
	require.NoError(t, err)

	ds, err := generateDatastore(ca, privateKey, caCertPEM, client.CertificateGracePeriod-time.Second)
	require.NoError(t, err)

	cf := &mockCredentialsFetcher{ds: ds}

	ctx := context.Background()
	tls, err := client.NewAutoRefreshingTransportCredentials(ctx, cf, "secret", "localhost")
	require.NoError(t, err)

	require.Equal(t, 1, cf.callCount, "Certificates are loaded once during initialization")

	server, client := net.Pipe()
	// Close pipe to avoid client getting stuck in an infinite reconnect loop
	err = server.Close()
	require.NoError(t, err)

	// Swap out certificates for fresh ones
	cf.ds, err = generateDatastore(ca, privateKey, caCertPEM, DistantFuture)
	require.NoError(t, err)

	_, _, err = tls.ClientHandshake(ctx, "", client)
	require.Error(t, err, "io: read/write on closed pipe")

	require.Equal(t, 2, cf.callCount, "Certificates are about to expiry and should be reloaded")

	_, _, err = tls.ClientHandshake(ctx, "", client)
	require.Error(t, err, "io: read/write on closed pipe")

	require.Equal(t, 2, cf.callCount, "Cached certificates are still valid and should not be reloaded")
}

var ca = &x509.Certificate{
	SerialNumber:          big.NewInt(2019),
	Subject:               pkix.Name{},
	NotBefore:             time.Now(),
	NotAfter:              time.Now().Add(DistantFuture),
	IsCA:                  true,
	BasicConstraintsValid: true,
}

func generateCA(privateKey *rsa.PrivateKey) ([]byte, error) {
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, err
	}

	caPEM := new(bytes.Buffer)
	err = pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	return caPEM.Bytes(), err
}

func generateDatastore(ca *x509.Certificate, privateKey *rsa.PrivateKey, caCertPEM []byte, validTime time.Duration) (credentialsmanager.DataStore, error) {
	ds := credentialsmanager.DataStore{}

	cert := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject:      pkix.Name{},
		DNSNames:     []string{"localhost"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(validTime),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &privateKey.PublicKey, privateKey)
	if err != nil {
		return ds, err
	}

	certPEM := new(bytes.Buffer)
	err = pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	if err != nil {
		return ds, err
	}

	certPrivKeyPEM := new(bytes.Buffer)

	err = pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if err != nil {
		return ds, err
	}

	ds.Crt = certPEM.Bytes()
	ds.Key = certPrivKeyPEM.Bytes()
	ds.CA = caCertPEM

	return ds, nil
}

type mockCredentialsFetcher struct {
	ds        credentialsmanager.DataStore
	callCount int
}

func (mock *mockCredentialsFetcher) GetDataStore(ctx context.Context, secretsName string) (*credentialsmanager.DataStore, error) {
	mock.callCount += 1
	return &mock.ds, nil
}

func loadTLSCredentials(ds credentialsmanager.DataStore) (credentials.TransportCredentials, error) {
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(ds.CA) {
		return nil, fmt.Errorf("failed to add CA certificate")
	}

	serverCert, err := tls.X509KeyPair(ds.Crt, ds.Key)
	if err != nil {
		return nil, err
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
		MinVersion:   tls.VersionTLS13,
	}

	return credentials.NewTLS(config), nil
}

type dummyAuthorizeServer struct {
	authorizeproto.UnimplementedAuthorizeServer
	failuresRemaining int
}

func (*dummyAuthorizeServer) LogClientState(context.Context, *authorizeproto.LogClientStateInput) (*common.Void, error) {
	return &common.Void{}, nil
}

func (srv *dummyAuthorizeServer) GetResource(context.Context, *authorizeproto.GetResourceInput) (*authorizeproto.GetResourceOutput, error) {
	if srv.failuresRemaining > 0 {
		srv.failuresRemaining--
		return nil, status.Errorf(codes.Canceled, "too slow")
	}

	return &authorizeproto.GetResourceOutput{
		Resource: &common.Origin{
			Id:       "",
			Type:     "",
			Provider: "",
		},
	}, nil
}

func parseRSAKey() (*rsa.PrivateKey, error) {
	pemBlock, _ := pem.Decode(rsaKey)
	k, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
	if err != nil {
		return nil, err
	}
	return k.(*rsa.PrivateKey), nil
}

type server struct {
	signal          chan struct{}
	tlsCredentials  credentials.TransportCredentials
	authorizeServer dummyAuthorizeServer
}

func newServer(tlsCredentials credentials.TransportCredentials, authorizeServer dummyAuthorizeServer) server {
	server := server{
		signal:          make(chan struct{}),
		tlsCredentials:  tlsCredentials,
		authorizeServer: authorizeServer,
	}

	return server
}

func (s *server) Start() error {
	var serverOpts []grpc.ServerOption
	serverOpts = append(serverOpts, grpc.Creds(s.tlsCredentials))

	grpcServer := grpc.NewServer(serverOpts...)
	authorizeproto.RegisterAuthorizeServer(grpcServer, &s.authorizeServer)

	lis, err := net.Listen("tcp", "localhost:10000")
	if err != nil {
		return err
	}

	go func() {
		//nolint:errcheck
		go grpcServer.Serve(lis)

		<-s.signal

		grpcServer.Stop()
		_ = lis.Close()
		s.signal <- struct{}{}
	}()

	return nil
}

func (s *server) Shutdown() {
	s.signal <- struct{}{}
	<-s.signal
}

func (s *server) Restart() error {
	s.Shutdown()
	return s.Start()
}

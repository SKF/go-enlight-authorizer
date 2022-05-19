package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type dataStore struct {
	CA  []byte `json:"ca"`
	Key []byte `json:"key"`
	Crt []byte `json:"crt"`
}

func getSecret(ctx context.Context, sess *session.Session, secretsName string, out interface{}) (err error) {
	// credentials - default
	svc := secretsmanager.New(sess)
	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretsName),
		VersionStage: aws.String("AWSCURRENT"),
	}

	result, err := svc.GetSecretValue(input)
	if err != nil {
		err = fmt.Errorf("failed to get secret value from '%s': %w", secretsName, err)
		return
	}

	if err = json.Unmarshal([]byte(*result.SecretString), out); err != nil {
		err = fmt.Errorf("failed to unmarshal secret from '%s': %w", secretsName, err)
	}

	return err
}

func getCredentialOption(ctx context.Context, sess *session.Session, host, secretKeyName string) (grpc.DialOption, error) {
	var clientCert dataStore
	if err := getSecret(ctx, sess, secretKeyName, &clientCert); err != nil {
		panic(err)
	}

	return withTransportCredentialsPEM(
		host,
		clientCert.Crt, clientCert.Key, clientCert.CA,
	)
}

func GetSecretKeyName(service, stage string) string {
	return fmt.Sprintf("authorize/%s/grpc/client/%s", stage, service)
}

func GetSecretKeyArn(accountId, region, service, stage string) string {
	return fmt.Sprintf("arn:aws:secretsmanager:%s:%s:secret:%s", region, accountId, GetSecretKeyName(service, stage))
}

func withTransportCredentialsPEM(serverName string, certPEMBlock, keyPEMBlock, caPEMBlock []byte) (opt grpc.DialOption, err error) {
	certificate, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
	if err != nil {
		err = fmt.Errorf("failed to load client certs, %+v", err)
		return
	}

	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM(caPEMBlock)
	if !ok {
		err = errors.New("failed to append certs")
		return
	}

	transportCreds := credentials.NewTLS(&tls.Config{
		ServerName:   serverName,
		Certificates: []tls.Certificate{certificate},
		RootCAs:      certPool,
	})

	opt = grpc.WithTransportCredentials(transportCreds)
	return
}

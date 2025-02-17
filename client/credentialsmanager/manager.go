package credentialsmanager

import (
	"context"
	"encoding/json"

	"github.com/SKF/go-utility/v2/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/pkg/errors"
)

type SMAPI interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}
type DataStore struct {
	CA  []byte `json:"ca"`
	Key []byte `json:"key"`
	Crt []byte `json:"crt"`
}

type credentialsManager struct {
	sm SMAPI
}

type CredentialsFetcher interface {
	GetDataStore(ctx context.Context, secretsName string) (*DataStore, error)
}

func New(sm SMAPI) *credentialsManager {
	return &credentialsManager{sm: sm}
}

func (cm *credentialsManager) GetDataStore(ctx context.Context, secretsName string) (*DataStore, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretsName),
		VersionStage: aws.String("AWSCURRENT"),
	}

	logger := log.
		WithField("secretsName", secretsName).
		WithField("credentialsManager", "V2")

	result, err := cm.sm.GetSecretValue(ctx, input)
	if err != nil {
		logger.WithTracing(ctx).WithError(err).
			Error("failed to get secrets")
		err = errors.Wrapf(err, "failed to get secret value from '%s'", secretsName)
		return nil, err
	}

	var out DataStore

	if err = json.Unmarshal([]byte(*result.SecretString), &out); err != nil {
		logger.WithTracing(ctx).WithError(err).
			Error("failed to unmarshal secret")
		err = errors.Wrapf(err, "failed to unmarshal secret from '%s'", secretsName)
		return nil, err
	}

	return &out, err
}

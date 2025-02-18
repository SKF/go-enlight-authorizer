package client

import "fmt"

func GetSecretKeyName(service, stage string) string {
	return fmt.Sprintf("authorize/%s/grpc/client/%s", stage, service)
}

func GetSecretKeyArn(accountId, region, service, stage string) string {
	return fmt.Sprintf("arn:aws:secretsmanager:%s:%s:secret:%s", region, accountId, GetSecretKeyName(service, stage))
}

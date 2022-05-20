package client

import (
	"context"
	"os"
	"time"

	"github.com/SKF/go-enlight-authorizer/interceptors/reconnect"
	"github.com/SKF/go-utility/v2/log"
	authorizeApi "github.com/SKF/proto/v2/authorize"
	"github.com/SKF/proto/v2/common"
	"github.com/aws/aws-sdk-go/aws/session"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/resolver"
)

const defaultServiceConfig = `{
	"loadBalancingConfig": [
		{ "round_robin": {} }
	]
}
`

type client struct {
	conn           *grpc.ClientConn
	api            authorizeApi.AuthorizeClient
	requestTimeout time.Duration
}

type AuthorizeClient interface {
	Dial(ctx context.Context, host, port string, opts ...grpc.DialOption) error
	DialUsingCredentials(ctx context.Context, sess *session.Session, host, port, secretKey string, opts ...grpc.DialOption) error
	SetRequestTimeout(d time.Duration)
	DeepPing(ctx context.Context) error
	Close() error

	IsAuthorized(ctx context.Context, userID, action string, resource *common.Origin) (bool, error)
	IsAuthorizedBulk(ctx context.Context, userID, action string, reqResources []*common.Origin) ([]*common.Origin, []bool, error)
	IsAuthorizedByEndpoint(ctx context.Context, api, method, endpoint, userID string) (bool, error)
	IsAuthorizedWithReason(ctx context.Context, userID, action string, resource *common.Origin) (bool, string, error)

	AddResource(ctx context.Context, resource *common.Origin) error
	GetResource(ctx context.Context, id, originType string) (*common.Origin, error)
	AddResources(ctx context.Context, resources []*common.Origin) error
	RemoveResource(ctx context.Context, resource *common.Origin) error
	RemoveResources(ctx context.Context, resources []*common.Origin) error

	GetResourcesWithActionsAccess(ctx context.Context, actions []string, resourceType string, resource *common.Origin) ([]*common.Origin, error)
	GetResourcesByUserAction(ctx context.Context, userID, actionName, resourceType string) ([]*common.Origin, error)
	GetResourcesByType(ctx context.Context, resourceType string) (resources []*common.Origin, err error)
	GetResourcesByOriginAndType(ctx context.Context, resource *common.Origin, resourceType string, depth int32) (resources []*common.Origin, err error)

	GetResourceParents(ctx context.Context, resource *common.Origin, parentOriginType string) (resources []*common.Origin, err error)
	GetResourceChildren(ctx context.Context, resource *common.Origin, childOriginType string) (resources []*common.Origin, err error)

	GetUserIDsWithAccessToResource(ctx context.Context, resource *common.Origin) (resources []string, err error)

	AddResourceRelation(ctx context.Context, resource, parent *common.Origin) error
	AddResourceRelations(ctx context.Context, resources *authorizeApi.AddResourceRelationsInput) error
	RemoveResourceRelation(ctx context.Context, resource, parent *common.Origin) error
	RemoveResourceRelations(ctx context.Context, resources *authorizeApi.RemoveResourceRelationsInput) error

	ApplyUserAction(ctx context.Context, userID, action string, resource *common.Origin) error
	ApplyRolesForUserOnResources(ctx context.Context, userID string, roles []string, resources []*common.Origin) error
	RemoveUserAction(ctx context.Context, userID, action string, resource *common.Origin) error
	GetActionsByUserRole(ctx context.Context, userRole string) ([]*authorizeApi.Action, error)
	GetResourcesAndActionsByUser(ctx context.Context, userID string) ([]*authorizeApi.ActionResource, error)
	GetResourcesAndActionsByUserAndResource(ctx context.Context, userID string, resource *common.Origin) ([]*authorizeApi.ActionResource, error)
	AddAction(ctx context.Context, action *authorizeApi.Action) error
	RemoveAction(ctx context.Context, name string) error
	GetAction(ctx context.Context, name string) (*authorizeApi.Action, error)
	GetAllActions(ctx context.Context) ([]*authorizeApi.Action, error)
	GetUserActions(ctx context.Context, userID string) ([]*authorizeApi.Action, error)

	AddUserRole(ctx context.Context, role *authorizeApi.UserRole) error
	GetUserRole(ctx context.Context, roleName string) (*authorizeApi.UserRole, error)
	RemoveUserRole(ctx context.Context, roleName string) error
}

func CreateClient() AuthorizeClient {
	return &client{
		requestTimeout: 60 * time.Second,
	}
}

// Dial creates a client connection to the given host with context (for timeout and transaction id)
func (c *client) Dial(ctx context.Context, host, port string, opts ...grpc.DialOption) (err error) {
	resolver.SetDefaultScheme("dns")
	opts = append(opts, grpc.WithDefaultServiceConfig(defaultServiceConfig))

	conn, err := grpc.DialContext(ctx, host+":"+port, opts...)
	if err != nil {
		return
	}

	c.conn = conn
	c.api = authorizeApi.NewAuthorizeClient(conn)
	err = c.logClientState(ctx, "opening connection")
	return
}

// DialUsingCredentials creates a client connection to the given host with context (for timeout and transaction id)
func (c *client) DialUsingCredentials(ctx context.Context, sess *session.Session, host, port, secretKey string, opts ...grpc.DialOption) error {
	resolver.SetDefaultScheme("dns")
	opts = append(opts, grpc.WithDefaultServiceConfig(defaultServiceConfig))

	var newClientConn reconnect.NewConnectionFunc
	newClientConn = func(invokerCtx context.Context, invokerConn *grpc.ClientConn, invokerOptions ...grpc.CallOption) (context.Context, *grpc.ClientConn, []grpc.CallOption, error) {
		credOpt, err := getCredentialOption(invokerCtx, sess, host, secretKey)
		if err != nil {
			log.WithTracing(invokerCtx).WithError(err).Error("Failed to get credential options")
			return invokerCtx, invokerConn, invokerOptions, err
		}

		dialOptsReconnectRetry := reconnectRetryInterceptor(newClientConn)

		dialOpts := append(opts, credOpt, dialOptsReconnectRetry, grpc.WithBlock())
		newConn, err := grpc.DialContext(invokerCtx, host+":"+port, dialOpts...)
		if err != nil {
			log.WithTracing(invokerCtx).WithError(err).Error("Failed to dial context")
			return invokerCtx, invokerConn, invokerOptions, err
		}
		_ = invokerConn.Close()

		c.conn = newConn
		c.api = authorizeApi.NewAuthorizeClient(c.conn)
		return invokerCtx, c.conn, invokerOptions, err
	}

	opt, err := getCredentialOption(ctx, sess, host, secretKey)
	if err != nil {
		return err
	}

	dialOptsReconnectRetry := reconnectRetryInterceptor(newClientConn)
	newOpts := append(opts, opt, dialOptsReconnectRetry)

	conn, err := grpc.DialContext(ctx, host+":"+port, newOpts...)
	if err != nil {
		return err
	}

	c.conn = conn
	c.api = authorizeApi.NewAuthorizeClient(c.conn)

	err = c.logClientState(ctx, "opening connection")
	return err
}

func reconnectRetryInterceptor(newClientConn reconnect.NewConnectionFunc) grpc.DialOption {
	retryIC := grpc_retry.UnaryClientInterceptor(
		grpc_retry.WithBackoff(grpc_retry.BackoffLinear(100*time.Millisecond)),
		grpc_retry.WithCodes(codes.Unavailable, codes.ResourceExhausted, codes.Aborted),
	)

	reconnectIC := reconnect.UnaryInterceptor(
		reconnect.WithNewConnection(newClientConn),
	)

	dialOptsReconnectRetry := grpc.WithChainUnaryInterceptor(reconnectIC, retryIC) // first one is outer, being called last
	return dialOptsReconnectRetry
}

func (c *client) Close() (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.requestTimeout)
	defer cancel()
	err = c.logClientState(ctx, "closing connection")
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *client) DeepPing(ctx context.Context) error {
	_, err := c.api.DeepPing(ctx, &common.Void{})
	return err
}

func (c *client) logClientState(ctx context.Context, state string) error {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
	}
	_, err = c.api.LogClientState(ctx, &authorizeApi.LogClientStateInput{
		State:    state,
		Hostname: hostname,
	})
	return err
}

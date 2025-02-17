package client

import (
	"context"
	_ "embed"
	"time"

	"github.com/SKF/go-enlight-authorizer/client/credentialsmanager"
	authorizeApi "github.com/SKF/proto/v2/authorize"
	"github.com/SKF/proto/v2/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
)

//go:embed service_config.json
var defaultServiceConfig string

type client struct {
	conn           *grpc.ClientConn
	api            authorizeApi.AuthorizeClient
	requestTimeout time.Duration
}

type AuthorizeClient interface {
	Dial(ctx context.Context, host, port string, opts ...grpc.DialOption) error
	DialUsingCredentialsManager(ctx context.Context, cf credentialsmanager.CredentialsFetcher, host, port, secretKey string, opts ...grpc.DialOption) error
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
func (c *client) Dial(ctx context.Context, host, port string, opts ...grpc.DialOption) error {
	resolver.SetDefaultScheme("dns")
	opts = append(opts, grpc.WithDefaultServiceConfig(defaultServiceConfig))

	conn, err := grpc.NewClient(host+":"+port, opts...)
	if err != nil {
		return err
	}

	c.conn = conn
	c.api = authorizeApi.NewAuthorizeClient(conn)

	return nil
}

// DialUsingCredentials creates a client connection to the given host with context (for timeout and transaction id)
func (c *client) DialUsingCredentialsManager(ctx context.Context, cf credentialsmanager.CredentialsFetcher, host, port, secretKey string, opts ...grpc.DialOption) error {
	resolver.SetDefaultScheme("dns")
	opts = append([]grpc.DialOption{grpc.WithDefaultServiceConfig(defaultServiceConfig)}, opts...)

	opt, err := getCredentialOption(ctx, cf, host, secretKey)
	if err != nil {
		return err
	}

	newOpts := append(opts, opt, withDefaultRequestTimeout(c.requestTimeout))

	conn, err := grpc.NewClient(host+":"+port, newOpts...)
	if err != nil {
		return err
	}

	c.conn = conn
	c.api = authorizeApi.NewAuthorizeClient(c.conn)

	return nil
}

func (c *client) Close() error {
	if c.conn == nil {
		return nil
	}

	return c.conn.Close()
}

func (c *client) DeepPing(ctx context.Context) error {
	_, err := c.api.DeepPing(ctx, &common.Void{})
	return err
}

func withDefaultRequestTimeout(requestTimeout time.Duration) grpc.DialOption {
	return grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		var cancel context.CancelFunc

		if _, ok := ctx.Deadline(); !ok {
			ctx, cancel = context.WithTimeout(ctx, requestTimeout)
			defer cancel()
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	})
}

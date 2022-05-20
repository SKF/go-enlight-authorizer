package mock

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	authorize "github.com/SKF/go-enlight-authorizer/client"
	grpcapi "github.com/SKF/proto/v2/authorize"
	"github.com/SKF/proto/v2/common"
)

type Client struct {
	mock.Mock
}

func Create() *Client { // nolint: golint
	return new(Client)
}

var _ authorize.AuthorizeClient = &Client{}

func (mock *Client) SetRequestTimeout(d time.Duration) {
	mock.Mock.Called(d)
}

func (mock *Client) Dial(ctx context.Context, host, port string, opts ...grpc.DialOption) error {
	args := mock.Mock.Called(ctx, host, port, opts)
	return args.Error(0)
}

func (mock *Client) DialUsingCredentials(ctx context.Context, sess *session.Session, host, port, secretKey string, opts ...grpc.DialOption) error {
	args := mock.Mock.Called(ctx, sess, host, port, secretKey, opts)
	return args.Error(0)
}

func (mock *Client) Close() error {
	args := mock.Mock.Called()
	return args.Error(0)
}

func (mock *Client) DeepPing(ctx context.Context) error {
	args := mock.Mock.Called(ctx)
	return args.Error(0)
}

func (mock *Client) IsAuthorized(ctx context.Context, userID, action string, resource *common.Origin) (bool, error) {
	args := mock.Mock.Called(ctx, userID, action, resource)
	return args.Bool(0), args.Error(1)
}

func (mock *Client) IsAuthorizedBulk(ctx context.Context, userID, action string, resources []*common.Origin) ([]*common.Origin, []bool, error) {
	args := mock.Mock.Called(userID, action, resources)
	return args.Get(0).([]*common.Origin), args.Get(1).([]bool), args.Error(2)
}

func (mock *Client) IsAuthorizedByEndpoint(ctx context.Context, api, method, endpoint, userID string) (bool, error) {
	args := mock.Mock.Called(ctx, api, method, endpoint, userID)
	return args.Bool(0), args.Error(1)
}

func (mock *Client) GetResourcesWithActionsAccess(ctx context.Context, actions []string, resourceType string, resource *common.Origin) ([]*common.Origin, error) {
	args := mock.Mock.Called(ctx, actions, resourceType, resource)
	return args.Get(0).([]*common.Origin), args.Error(1)
}

func (mock *Client) GetResourcesByUserAction(ctx context.Context, userID, actionName, resourceType string) ([]*common.Origin, error) {
	args := mock.Mock.Called(ctx, userID, actionName, resourceType)
	return args.Get(0).([]*common.Origin), args.Error(1)
}

func (mock *Client) GetResourcesByType(ctx context.Context, resourceType string) (resources []*common.Origin, err error) {
	args := mock.Mock.Called(ctx, resourceType)
	return args.Get(0).([]*common.Origin), args.Error(1)
}

func (mock *Client) GetResourceParents(ctx context.Context, resource *common.Origin, parentOriginType string) (resources []*common.Origin, err error) {
	args := mock.Mock.Called(ctx, resource, parentOriginType)
	return args.Get(0).([]*common.Origin), args.Error(1)
}

func (mock *Client) GetResourceChildren(ctx context.Context, resource *common.Origin, childOriginType string) (resources []*common.Origin, err error) {
	args := mock.Mock.Called(ctx, resource, childOriginType)
	return args.Get(0).([]*common.Origin), args.Error(1)
}

func (mock *Client) AddResource(ctx context.Context, resource *common.Origin) error {
	args := mock.Mock.Called(ctx, resource)
	return args.Error(0)
}

func (mock *Client) GetResource(ctx context.Context, id, originType string) (*common.Origin, error) {
	args := mock.Mock.Called(id, originType)
	return args.Get(0).(*common.Origin), args.Error(1)
}

func (mock *Client) AddResourceRelation(ctx context.Context, resource, parent *common.Origin) error {
	args := mock.Mock.Called(ctx, resource, parent)
	return args.Error(0)
}

func (mock *Client) RemoveResourceRelation(ctx context.Context, resource, parent *common.Origin) error {
	args := mock.Mock.Called(ctx, resource, parent)
	return args.Error(0)
}

func (mock *Client) RemoveResource(ctx context.Context, resource *common.Origin) error {
	args := mock.Mock.Called(ctx, resource)
	return args.Error(0)
}

func (mock *Client) ApplyUserAction(ctx context.Context, userID, action string, resource *common.Origin) error {
	args := mock.Mock.Called(ctx, userID, action, resource)
	return args.Error(0)
}

func (mock *Client) ApplyRolesForUserOnResources(ctx context.Context, userID string, roles []string, resources []*common.Origin) error {
	args := mock.Mock.Called(ctx, userID, roles, resources)
	return args.Error(0)
}

func (mock *Client) RemoveUserAction(ctx context.Context, userID, action string, resource *common.Origin) error {
	args := mock.Mock.Called(ctx, userID, action, resource)
	return args.Error(0)
}

func (mock *Client) GetResourcesByOriginAndType(ctx context.Context, resource *common.Origin, resourceType string, depth int32) (resources []*common.Origin, err error) {
	args := mock.Mock.Called(ctx, resource, resourceType, depth)
	return args.Get(0).([]*common.Origin), args.Error(1)
}

func (mock *Client) GetUserIDsWithAccessToResource(ctx context.Context, resource *common.Origin) (resources []string, err error) {
	args := mock.Mock.Called(ctx, resource)
	return args.Get(0).([]string), args.Error(1)
}

func (mock *Client) AddResources(ctx context.Context, resources []*common.Origin) error {
	args := mock.Mock.Called(ctx, resources)
	return args.Error(0)
}

func (mock *Client) RemoveResources(ctx context.Context, resources []*common.Origin) error {
	args := mock.Mock.Called(ctx, resources)
	return args.Error(0)
}

func (mock *Client) AddResourceRelations(ctx context.Context, resources *grpcapi.AddResourceRelationsInput) error {
	args := mock.Mock.Called(ctx, resources)
	return args.Error(0)
}

func (mock *Client) RemoveResourceRelations(ctx context.Context, resources *grpcapi.RemoveResourceRelationsInput) error {
	args := mock.Mock.Called(ctx, resources)
	return args.Error(0)
}

func (mock *Client) GetActionsByUserRole(ctx context.Context, userRole string) ([]*grpcapi.Action, error) {
	args := mock.Mock.Called(ctx, userRole)
	return args.Get(0).([]*grpcapi.Action), args.Error(1)
}

func (mock *Client) GetResourcesAndActionsByUser(ctx context.Context, userID string) ([]*grpcapi.ActionResource, error) {
	args := mock.Mock.Called(ctx, userID)
	return args.Get(0).([]*grpcapi.ActionResource), args.Error(1)
}

func (mock *Client) GetResourcesAndActionsByUserAndResource(ctx context.Context, userID string, resource *common.Origin) ([]*grpcapi.ActionResource, error) {
	args := mock.Mock.Called(ctx, userID, resource)
	return args.Get(0).([]*grpcapi.ActionResource), args.Error(1)
}

func (mock *Client) AddAction(ctx context.Context, action *grpcapi.Action) error {
	args := mock.Mock.Called(ctx, action)
	return args.Error(0)
}

func (mock *Client) RemoveAction(ctx context.Context, name string) error {
	args := mock.Mock.Called(ctx, name)
	return args.Error(0)
}

func (mock *Client) GetAction(ctx context.Context, name string) (*grpcapi.Action, error) {
	args := mock.Mock.Called(ctx, name)
	return args.Get(0).(*grpcapi.Action), args.Error(1)
}

func (mock *Client) GetAllActions(ctx context.Context) ([]*grpcapi.Action, error) {
	args := mock.Mock.Called(ctx)
	return args.Get(0).([]*grpcapi.Action), args.Error(1)
}

func (mock *Client) GetUserActions(ctx context.Context, userID string) ([]*grpcapi.Action, error) {
	args := mock.Mock.Called(ctx, userID)
	return args.Get(0).([]*grpcapi.Action), args.Error(1)
}

func (mock *Client) AddUserRole(ctx context.Context, role *grpcapi.UserRole) error {
	args := mock.Mock.Called(ctx, role)
	return args.Error(0)
}

func (mock *Client) GetUserRole(ctx context.Context, roleName string) (*grpcapi.UserRole, error) {
	args := mock.Mock.Called(ctx, roleName)
	return args.Get(0).(*grpcapi.UserRole), args.Error(1)
}

func (mock *Client) RemoveUserRole(ctx context.Context, roleName string) error {
	args := mock.Mock.Called(ctx, roleName)
	return args.Error(0)
}

func (mock *Client) IsAuthorizedWithReason(ctx context.Context, userID, action string, resource *common.Origin) (bool, string, error) {
	args := mock.Mock.Called(ctx, userID, action, resource)
	return args.Bool(0), args.String(1), args.Error(2)
}

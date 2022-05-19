package client

import (
	"context"
	"errors"
	"fmt"
	"time"

	authorizeApi "github.com/SKF/proto/v2/authorize"
	"github.com/SKF/proto/v2/common"
)

const REQUEST_LENGTH_LIMIT = 1000

func requestLengthLimit(requestLength int) error {
	if requestLength > REQUEST_LENGTH_LIMIT {
		return fmt.Errorf("request length limit exceeded. max: %d actual: %d", REQUEST_LENGTH_LIMIT, requestLength)
	}
	return nil
}

func (c *client) SetRequestTimeout(d time.Duration) {
	c.requestTimeout = d
}

func (c *client) IsAuthorized(ctx context.Context, userID, action string, resource *common.Origin) (bool, error) {
	result, err := c.api.IsAuthorized(ctx, &authorizeApi.IsAuthorizedInput{
		UserId:   userID,
		Action:   action,
		Resource: resource,
	})
	if err != nil {
		return false, err
	}

	return result.Ok, err
}

func (c *client) IsAuthorizedBulk(ctx context.Context, userID, action string, resourcesInput []*common.Origin) ([]*common.Origin, []bool, error) {
	if err := requestLengthLimit(len(resourcesInput)); err != nil {
		return nil, nil, err
	}

	results, err := c.api.IsAuthorizedBulk(ctx, &authorizeApi.IsAuthorizedBulkInput{
		UserId:    userID,
		Action:    action,
		Resources: resourcesInput,
	})
	if err != nil {
		return nil, nil, err
	}

	responses := results.GetResponses()
	resources := make([]*common.Origin, len(responses))
	oks := make([]bool, len(responses))

	for i := range responses {
		resource := responses[i].GetResource()
		// If running against an old server which doesn't set the resource
		if resource == nil {
			resource = &common.Origin{
				Id:       responses[i].GetResourceId(), //nolint: staticcheck
				Type:     "",
				Provider: "",
			}
		}
		resources[i] = resource
		oks[i] = responses[i].GetOk()
	}

	return resources, oks, err
}

func (c *client) IsAuthorizedByEndpoint(ctx context.Context, api, method, endpoint, userID string) (bool, error) {
	result, err := c.api.IsAuthorizedByEndpoint(ctx, &authorizeApi.IsAuthorizedByEndpointInput{
		Api:      api,
		Method:   method,
		Endpoint: endpoint,
		UserId:   userID,
	})
	if err != nil {
		return false, err
	}

	return result.Ok, err
}

func (c *client) AddResource(ctx context.Context, resource *common.Origin) error {
	_, err := c.api.AddResource(ctx, &authorizeApi.AddResourceInput{
		Resource: resource,
	})
	return err
}

func (c *client) GetResource(ctx context.Context, id, originType string) (*common.Origin, error) {
	input := authorizeApi.GetResourceInput{
		Id:         id,
		OriginType: originType,
	}
	resource, err := c.api.GetResource(ctx, &input)
	if err != nil {
		return nil, err
	}

	return resource.Resource, err
}

func (c *client) AddResources(ctx context.Context, resources []*common.Origin) error {
	if err := requestLengthLimit(len(resources)); err != nil {
		return err
	}

	_, err := c.api.AddResources(ctx, &authorizeApi.AddResourcesInput{
		Resource: resources,
	})
	return err
}

func (c *client) RemoveResource(ctx context.Context, resource *common.Origin) error {
	_, err := c.api.RemoveResource(ctx, &authorizeApi.RemoveResourceInput{
		Resource: resource,
	})
	return err
}

func (c *client) RemoveResources(ctx context.Context, resources []*common.Origin) error {
	if err := requestLengthLimit(len(resources)); err != nil {
		return err
	}

	_, err := c.api.RemoveResources(ctx, &authorizeApi.RemoveResourcesInput{
		Resource: resources,
	})
	return err
}

func (c *client) GetResourcesByUserAction(ctx context.Context, userID, actionName, resourceType string) (resources []*common.Origin, err error) {
	input := authorizeApi.GetResourcesByUserActionInput{
		UserId:       userID,
		Action:       actionName,
		ResourceType: resourceType,
	}
	output, err := c.api.GetResourcesByUserAction(ctx, &input)
	if err != nil {
		return
	}
	if output != nil {
		resources = output.Resources
	}
	return
}

func (c *client) GetResourcesWithActionsAccess(ctx context.Context, actions []string, resourceType string, resource *common.Origin) (resources []*common.Origin, err error) {
	input := authorizeApi.GetResourcesWithActionsAccessInput{
		Actions:      actions,
		ResourceType: resourceType,
		Resource:     resource,
	}
	output, err := c.api.GetResourcesWithActionsAccess(ctx, &input)
	if err != nil {
		return
	}
	if output != nil {
		resources = output.Resources
	}
	return
}

func (c *client) GetResourcesByType(ctx context.Context, resourceType string) (resources []*common.Origin, err error) {
	input := authorizeApi.GetResourcesByTypeInput{ResourceType: resourceType}
	output, err := c.api.GetResourcesByType(ctx, &input)
	if err != nil {
		return
	}
	if output != nil {
		resources = output.Resources
	}
	return
}

func (c *client) AddResourceRelation(ctx context.Context, resource, parent *common.Origin) error {
	_, err := c.api.AddResourceRelation(ctx, &authorizeApi.AddResourceRelationInput{
		Resource: resource,
		Parent:   parent,
	})
	return err
}

func (c *client) AddResourceRelations(ctx context.Context, resources *authorizeApi.AddResourceRelationsInput) error {
	if err := requestLengthLimit(len(resources.Relation)); err != nil {
		return err
	}

	_, err := c.api.AddResourceRelations(ctx, resources)
	return err
}

func (c *client) RemoveResourceRelation(ctx context.Context, resource, parent *common.Origin) error {
	_, err := c.api.RemoveResourceRelation(ctx, &authorizeApi.RemoveResourceRelationInput{
		Resource: resource,
		Parent:   parent,
	})
	return err
}

func (c *client) RemoveResourceRelations(ctx context.Context, resources *authorizeApi.RemoveResourceRelationsInput) error {
	if err := requestLengthLimit(len(resources.Relation)); err != nil {
		return err
	}

	_, err := c.api.RemoveResourceRelations(ctx, resources)
	return err
}

func (c *client) ApplyUserAction(ctx context.Context, userID, action string, resource *common.Origin) error {
	_, err := c.api.ApplyUserAction(ctx, &authorizeApi.ApplyUserActionInput{
		UserId:   userID,
		Action:   action,
		Resource: resource,
	})
	return err
}

func (c *client) ApplyRolesForUserOnResources(ctx context.Context, userID string, roles []string, resources []*common.Origin) error {
	_, err := c.api.ApplyRolesForUserOnResources(ctx, &authorizeApi.ApplyRolesForUserOnResourcesInput{
		UserId:    userID,
		Roles:     roles,
		Resources: resources,
	})

	return err
}

func (c *client) RemoveUserAction(ctx context.Context, userID, action string, resource *common.Origin) error {
	_, err := c.api.RemoveUserAction(ctx, &authorizeApi.RemoveUserActionInput{
		UserId:   userID,
		Action:   action,
		Resource: resource,
	})
	return err
}

func (c *client) GetResourcesByOriginAndType(ctx context.Context, resource *common.Origin, resourceType string, depth int32) (resources []*common.Origin, err error) {
	input := authorizeApi.GetResourcesByOriginAndTypeInput{ResourceType: resourceType, Resource: resource, Depth: depth}
	output, err := c.api.GetResourcesByOriginAndType(ctx, &input)
	if err != nil {
		return
	}
	if output != nil {
		resources = output.Resources
	}
	return
}

func (c *client) GetResourceParents(ctx context.Context, resource *common.Origin, parentOriginType string) (resources []*common.Origin, err error) {
	input := authorizeApi.GetResourceParentsInput{ParentOriginType: parentOriginType, Resource: resource}
	output, err := c.api.GetResourceParents(ctx, &input)
	if err != nil {
		return
	}
	if output != nil {
		resources = output.Resources
	}
	return
}

func (c *client) GetResourceChildren(ctx context.Context, resource *common.Origin, childOriginType string) (resources []*common.Origin, err error) {
	input := authorizeApi.GetResourceChildrenInput{ChildOriginType: childOriginType, Resource: resource}
	output, err := c.api.GetResourceChildren(ctx, &input)
	if err != nil {
		return
	}
	if output != nil {
		resources = output.Resources
	}
	return
}

func (c *client) GetUserIDsWithAccessToResource(ctx context.Context, resource *common.Origin) (userIds []string, err error) {
	input := authorizeApi.GetUserIDsWithAccessToResourceInput{Resource: resource}
	output, err := c.api.GetUserIDsWithAccessToResource(ctx, &input)
	if err != nil {
		return
	}
	if output != nil {
		userIds = output.UserIds
	}
	return
}

func (c *client) GetActionsByUserRole(ctx context.Context, userRole string) (actions []*authorizeApi.Action, err error) {
	input := authorizeApi.GetActionsByUserRoleInput{UserRole: userRole}
	output, err := c.api.GetActionsByUserRole(ctx, &input)
	if err != nil {
		return
	}

	if output != nil {
		actions = output.Actions
	}

	return
}

func (c *client) GetResourcesAndActionsByUser(ctx context.Context, userID string) (actionResources []*authorizeApi.ActionResource, err error) {
	input := authorizeApi.GetResourcesAndActionsByUserInput{UserId: userID}
	output, err := c.api.GetResourcesAndActionsByUser(ctx, &input)
	if err != nil {
		return
	}

	if output != nil {
		actionResources = output.Data
	}

	return
}

func (c *client) GetResourcesAndActionsByUserAndResource(ctx context.Context, userID string, resource *common.Origin) (actionResources []*authorizeApi.ActionResource, err error) {
	input := authorizeApi.GetResourcesAndActionsByUserAndResourceInput{UserId: userID, Resource: resource}
	output, err := c.api.GetResourcesAndActionsByUserAndResource(ctx, &input)
	if err != nil {
		return
	}

	if output != nil {
		actionResources = output.Data
	}

	return
}

func (c *client) AddAction(ctx context.Context, action *authorizeApi.Action) error {
	_, err := c.api.AddAction(ctx, &authorizeApi.AddActionInput{Action: action})
	return err
}

func (c *client) RemoveAction(ctx context.Context, name string) error {
	_, err := c.api.RemoveAction(ctx, &authorizeApi.RemoveActionInput{Name: name})
	return err
}

func (c *client) GetAction(ctx context.Context, name string) (actions *authorizeApi.Action, err error) {
	input := authorizeApi.GetActionInput{Name: name}
	action, err := c.api.GetAction(ctx, &input)
	if err != nil {
		return
	}
	return action.Action, err
}

func (c *client) GetAllActions(ctx context.Context) (actions []*authorizeApi.Action, err error) {
	allActions, err := c.api.GetAllActions(ctx, &common.Void{})
	if err != nil {
		return
	}
	if allActions != nil {
		actions = allActions.Actions
	}

	return
}

func (c *client) GetUserActions(ctx context.Context, userID string) (actions []*authorizeApi.Action, err error) {
	result, err := c.api.GetUserActions(ctx, &authorizeApi.GetUserActionsInput{
		UserId: userID,
	})

	if err != nil {
		return
	} else if result != nil {
		actions = result.Actions
	}

	return
}

func (c *client) AddUserRole(ctx context.Context, role *authorizeApi.UserRole) error {
	_, err := c.api.AddUserRole(ctx, role)
	return err
}

func (c *client) GetUserRole(ctx context.Context, roleName string) (role *authorizeApi.UserRole, err error) {
	result, err := c.api.GetUserRole(ctx, &authorizeApi.GetUserRoleInput{
		RoleName: roleName,
	})
	if err != nil {
		return
	}

	if result != nil {
		role = result
	} else {
		err = errors.New("No result")
	}
	return
}

func (c *client) RemoveUserRole(ctx context.Context, roleName string) error {
	_, err := c.api.RemoveUserRole(ctx, &authorizeApi.RemoveUserRoleInput{
		RoleName: roleName,
	})
	return err
}

func (c *client) IsAuthorizedWithReason(ctx context.Context, userID, action string, resource *common.Origin) (bool, string, error) {
	result, err := c.api.IsAuthorizedWithReason(ctx, &authorizeApi.IsAuthorizedInput{
		UserId:   userID,
		Action:   action,
		Resource: resource,
	})
	if err != nil || result == nil {
		return false, "error occurred ", err
	}

	return result.Ok, result.Reason, err
}

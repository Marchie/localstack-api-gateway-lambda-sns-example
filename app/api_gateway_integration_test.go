// +build integration

package app

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/apigateway/apigatewayiface"
	"github.com/pkg/errors"
)

func CreateAPIGatewayRestAPI(apiGatewayClient apigatewayiface.APIGatewayAPI, name string) (string, error) {
	createRestApiOutput, err := apiGatewayClient.CreateRestApi(&apigateway.CreateRestApiInput{
		Name: &name,
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed creating API Gateway REST API '%s'", name)
	}

	return *createRestApiOutput.Id, nil
}

func DeleteAPIGatewayRestAPI(apiGatewayClient apigatewayiface.APIGatewayAPI, restApiId string) error {
	if _, err := apiGatewayClient.DeleteRestApi(&apigateway.DeleteRestApiInput{
		RestApiId: &restApiId,
	}); err != nil {
		return errors.Wrapf(err, "failed deleting API Gateway REST API '%s'", restApiId)
	}

	return nil
}

func FindAPIGatewayParentResourceId(apiGatewayClient apigatewayiface.APIGatewayAPI, restApiId string) (string, error) {
	getResourcesOutput, err := apiGatewayClient.GetResources(&apigateway.GetResourcesInput{
		RestApiId: &restApiId,
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed to get resources for API Gateway REST API '%s'", restApiId)
	}

	for _, item := range getResourcesOutput.Items {
		if *item.Path == "/" {
			return *item.Id, nil
		}
	}

	return "", fmt.Errorf("failed to find Parent Resource ID for API Gateway REST API '%s'", restApiId)
}

func CreateAPIGatewayResource(apiGatewayClient apigatewayiface.APIGatewayAPI, parentId, pathPart, restApiId string) (string, error) {
	createResourceOutput, err := apiGatewayClient.CreateResource(&apigateway.CreateResourceInput{
		ParentId:  &parentId,
		PathPart:  &pathPart,
		RestApiId: &restApiId,
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed creating API Gateway resource for parent '%s' on path '%s' for REST API '%s'", parentId, pathPart, restApiId)
	}

	return *createResourceOutput.Id, nil
}

func DeleteAPIGatewayResource(apiGatewayClient apigatewayiface.APIGatewayAPI, resourceId, restApiId string) error {
	if _, err := apiGatewayClient.DeleteResource(&apigateway.DeleteResourceInput{
		ResourceId: &resourceId,
		RestApiId:  &restApiId,
	}); err != nil {
		return errors.Wrapf(err, "failed deleting API Gateway resource '%s' for REST API '%s'", resourceId, restApiId)
	}

	return nil
}

func PutAPIGatewayMethod(apiGatewayClient apigatewayiface.APIGatewayAPI, httpMethod, resourceId, restApiId, authorizationType string) error {
	if _, err := apiGatewayClient.PutMethod(&apigateway.PutMethodInput{
		AuthorizationType: &authorizationType,
		HttpMethod:        &httpMethod,
		ResourceId:        &resourceId,
		RestApiId:         &restApiId,
	}); err != nil {
		return errors.Wrapf(err, "failed to put method '%s' for Resource ID '%s' and API Gateway REST API '%s'", httpMethod, resourceId, restApiId)
	}

	return nil
}

func DeleteAPIGatewayMethod(apiGatewayClient apigatewayiface.APIGatewayAPI, httpMethod, resourceId, restApiId string) error {
	if _, err := apiGatewayClient.DeleteMethod(&apigateway.DeleteMethodInput{
		HttpMethod: &httpMethod,
		ResourceId: &resourceId,
		RestApiId:  &restApiId,
	}); err != nil {
		return errors.Wrapf(err, "failed to delete method '%s' for Resource ID '%s' and API Gateway REST API '%s'", httpMethod, resourceId, restApiId)
	}

	return nil
}

func PutAPIGatewayIntegration(apiGatewayClient apigatewayiface.APIGatewayAPI, lambdaArn, httpMethod, resourceId, restApiId, integrationType, passThroughBehaviour, uri string) error {
	if _, err := apiGatewayClient.PutIntegration(&apigateway.PutIntegrationInput{
		HttpMethod:            &httpMethod,
		IntegrationHttpMethod: &httpMethod,
		PassthroughBehavior:   &passThroughBehaviour,
		ResourceId:            &resourceId,
		RestApiId:             &restApiId,
		Type:                  &integrationType,
		Uri:                   &uri,
	}); err != nil {
		return errors.Wrapf(err, "failed to put integration with Lambda ARN '%s' for method '%s' on Resource ID '%s' and API Gateway REST API '%s'", lambdaArn, httpMethod, resourceId, restApiId)
	}

	return nil
}

func DeleteAPIGatewayIntegration(apiGatewayClient apigatewayiface.APIGatewayAPI, httpMethod, resourceId, restApiId string) error {
	if _, err := apiGatewayClient.DeleteIntegration(&apigateway.DeleteIntegrationInput{
		HttpMethod: &httpMethod,
		ResourceId: &resourceId,
		RestApiId:  &restApiId,
	}); err != nil {
		return errors.Wrapf(err, "failed to delete integration for method '%s' on Resource ID '%s' and API Gateway REST API '%s'", httpMethod, resourceId, restApiId)
	}

	return nil
}

func CreateAPIGatewayDeployment(apiGatewayClient apigatewayiface.APIGatewayAPI, restApiId, stage string) (string, error) {
	createDeploymentOutput, err := apiGatewayClient.CreateDeployment(&apigateway.CreateDeploymentInput{
		RestApiId: &restApiId,
		StageName: &stage,
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed to create API Gateway deployment for REST API '%s'", restApiId)
	}

	return *createDeploymentOutput.Id, nil
}

func DeleteAPIGatewayDeployment(apiGatewayClient apigatewayiface.APIGatewayAPI, deploymentId, restApiId string) error {
	if _, err := apiGatewayClient.DeleteDeployment(&apigateway.DeleteDeploymentInput{
		DeploymentId: &deploymentId,
		RestApiId:    &restApiId,
	}); err != nil {
		return errors.Wrapf(err, "failed to delete API Gateway deployment '%s' for REST API '%s'", deploymentId, restApiId)
	}

	return nil
}

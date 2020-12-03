// +build integration

package app

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/pkg/errors"
)

func CreateLambdaFunction(lambdaClient lambdaiface.LambdaAPI, functionCode []byte, functionName, handler, role, runtime string, memorySize, timeout int64, environment map[string]*string) (string, error) {
	if timeout < 1 || timeout > 900 {
		return "", errors.New("invalid timeout value: timeout must be between 1 and 900")
	}

	if memorySize < 128 || memorySize > 3008 || memorySize%64 > 0 {
		return "", errors.New("invalid memory size value: memory size must be between 128 and 3008 and be divisible by 64")
	}

	functionConfiguration, err := lambdaClient.CreateFunction(&lambda.CreateFunctionInput{
		Code: &lambda.FunctionCode{
			ZipFile: functionCode,
		},
		Environment: &lambda.Environment{
			Variables: environment,
		},
		FunctionName: &functionName,
		Handler:      &handler,
		MemorySize:   &memorySize,
		Role:         &role,
		Runtime:      &runtime,
		Timeout:      &timeout,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed creating Lambda function")
	}

	return *functionConfiguration.FunctionArn, nil
}

func DeleteLambdaFunction(lambdaClient lambdaiface.LambdaAPI, lambdaArn string) error {
	if _, err := lambdaClient.DeleteFunction(&lambda.DeleteFunctionInput{
		FunctionName: &lambdaArn,
	}); err != nil {
		return errors.Wrap(err, "")
	}

	return nil
}

func GetLambdaURI(awsRegion, lambdaArn string) string {
	return fmt.Sprintf("arn:aws:apigateway:%s:lambda:path/2015-03-31/functions/%s/invocations", awsRegion, lambdaArn)
}

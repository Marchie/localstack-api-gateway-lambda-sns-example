// +build integration

package app

import (
	"context"
	"errors"
	"github.com/Marchie/localstack-api-gateway-lambda-sns-example/pkg/repository"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/apigateway/apigatewayiface"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/gomodule/redigo/redis"
	"time"
)

func RedisServerIsAvailable(ctx context.Context, redisPool repository.RedisPooler, checkInterval time.Duration) error {
	for {
		select {
		case <-ctx.Done():
			return errors.New("timed out waiting for Redis server to be available")
		default:
			conn := redisPool.Get()
			if _, err := redis.String(conn.Do("PING")); err == nil {
				_ = conn.Close()
				return nil
			}
			time.Sleep(checkInterval)
		}
	}
}

func APIGatewayServiceIsAvailable(ctx context.Context, apiGatewayClient apigatewayiface.APIGatewayAPI, checkInterval time.Duration) error {
	for {
		select {
		case <-ctx.Done():
			return errors.New("timed out waiting for API Gateway service to be available")
		default:
			if _, err := apiGatewayClient.GetRestApis(&apigateway.GetRestApisInput{}); err == nil {
				return nil
			}
			time.Sleep(checkInterval)
		}
	}
}

func LambdaServiceIsAvailable(ctx context.Context, lambdaClient lambdaiface.LambdaAPI, checkInterval time.Duration) error {
	for {
		select {
		case <-ctx.Done():
			return errors.New("timed out waiting for Lambda service to be available")
		default:
			if _, err := lambdaClient.ListFunctions(&lambda.ListFunctionsInput{}); err == nil {
				return nil
			}
			time.Sleep(checkInterval)
		}
	}
}

func SNSServiceIsAvailable(ctx context.Context, snsClient snsiface.SNSAPI, checkInterval time.Duration) error {
	for {
		select {
		case <-ctx.Done():
			return errors.New("timed out waiting for Lambda service to be available")
		default:
			if _, err := snsClient.ListTopics(&sns.ListTopicsInput{}); err == nil {
				return nil
			}
			time.Sleep(checkInterval)
		}
	}
}

package app

import (
	"context"
	"github.com/Marchie/localstack-api-gateway-lambda-sns-example/pkg/repository"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/gomodule/redigo/redis"
	"net/http"
)

type App struct {
	pool        repository.RedisPooler
	snsClient   snsiface.SNSAPI
	snsTopicArn string
}

func New(pool repository.RedisPooler, sns snsiface.SNSAPI, snsTopicArn string) *App {
	return &App{
		pool:        pool,
		snsClient:   sns,
		snsTopicArn: snsTopicArn,
	}
}

func (a *App) Handler(ctx context.Context, event *events.APIGatewayProxyRequest) (res *events.APIGatewayProxyResponse, err error) {
	if event.Body == "" {
		return &events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
		}, nil
	}

	lc, _ := lambdacontext.FromContext(ctx)

	conn := a.pool.Get()
	defer func() {
		if connErr := conn.Close(); connErr != nil {
			err = connErr
		}
	}()

	if _, err := redis.String(conn.Do("SET", lc.AwsRequestID, event.Body)); err != nil {
		return nil, err
	}

	if _, err := a.snsClient.Publish(&sns.PublishInput{
		Message:  &lc.AwsRequestID,
		TopicArn: &a.snsTopicArn,
	}); err != nil {
		return nil, err
	}

	return &events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
	}, nil
}

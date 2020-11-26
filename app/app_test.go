package app

import (
	"context"
	"errors"
	mock_repository "github.com/Marchie/localstack-api-gateway-lambda-sns-example/mocks/pkg/repository"
	mock_snsiface "github.com/Marchie/localstack-api-gateway-lambda-sns-example/mocks/third-party/aws/aws-sdk-go/service/sns/snsiface"
	mock_redis "github.com/Marchie/localstack-api-gateway-lambda-sns-example/mocks/third-party/gomodule/redigo"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/golang/mock/gomock"
	"net/http"
	"testing"
)

func TestApp_Handler(t *testing.T) {
	t.Run(`given a configured App
when Handler is called with an APIGatewayProxyRequest that has an empty Body
then an APIGatewayProxyResponse is returned with a 400 status
and no key is stored in the configured Redis cache
and no messages are published to the configured SNS topic
and there is no error`, func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pool := mock_repository.NewMockRedisPooler(ctrl)
		snsClient := mock_snsiface.NewMockSNSAPI(ctrl)
		snsTopicArn := "arn:aws:sns:us-east-1:000000000000:mytopic"

		app := New(pool, snsClient, snsTopicArn)

		awsRequestId := "943ad105-7543-11e6-a9ac-65e093327849"

		ctx := lambdacontext.NewContext(context.Background(), &lambdacontext.LambdaContext{
			AwsRequestID: awsRequestId,
		})

		var event events.APIGatewayProxyRequest

		// When
		res, err := app.Handler(ctx, &event)

		// Then
		if err != nil {
			t.Error(err)
		}

		expStatusCode := http.StatusBadRequest
		if res.StatusCode != expStatusCode {
			t.Errorf("got %d, want %d", res.StatusCode, expStatusCode)
		}
	})

	t.Run(`given a configured App
when Handler is called with an APIGatewayProxyRequest that has an populated Body
then the Body is stored in the configured Redis cache
and the key the body was stored under is published to the configured SNS topic
and an APIGatewayProxyResponse is returned with a 200 status
and there is no error`, func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pool := mock_repository.NewMockRedisPooler(ctrl)
		snsClient := mock_snsiface.NewMockSNSAPI(ctrl)
		snsTopicArn := "arn:aws:sns:us-east-1:000000000000:mytopic"

		app := New(pool, snsClient, snsTopicArn)

		awsRequestId := "943ad105-7543-11e6-a9ac-65e093327849"

		ctx := lambdacontext.NewContext(context.Background(), &lambdacontext.LambdaContext{
			AwsRequestID: awsRequestId,
		})

		body := "foo"

		event := events.APIGatewayProxyRequest{
			Body: body,
		}

		conn := mock_redis.NewMockConn(ctrl)

		gomock.InOrder(
			pool.EXPECT().Get().Times(1).Return(conn),
			conn.EXPECT().Do("SET", awsRequestId, body).Times(1).Return("OK", nil),
			snsClient.EXPECT().Publish(&sns.PublishInput{
				Message:  &awsRequestId,
				TopicArn: &snsTopicArn,
			}).Times(1).Return(&sns.PublishOutput{}, nil),
			conn.EXPECT().Close().Times(1).Return(nil),
		)

		// When
		res, err := app.Handler(ctx, &event)

		// Then
		if err != nil {
			t.Error(err)
		}

		expStatusCode := http.StatusOK
		if res.StatusCode != expStatusCode {
			t.Errorf("got %d, want %d", res.StatusCode, expStatusCode)
		}
	})

	t.Run(`given a configured App
when Handler is called with an APIGatewayProxyRequest that has an populated Body
and an error occurs executing the Redis command
then there is no response
and there is no error`, func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pool := mock_repository.NewMockRedisPooler(ctrl)
		snsClient := mock_snsiface.NewMockSNSAPI(ctrl)
		snsTopicArn := "arn:aws:sns:us-east-1:000000000000:mytopic"

		app := New(pool, snsClient, snsTopicArn)

		awsRequestId := "943ad105-7543-11e6-a9ac-65e093327849"

		ctx := lambdacontext.NewContext(context.Background(), &lambdacontext.LambdaContext{
			AwsRequestID: awsRequestId,
		})

		body := "foo"

		event := events.APIGatewayProxyRequest{
			Body: body,
		}

		conn := mock_redis.NewMockConn(ctrl)

		redisErr := errors.New("FUBAR")

		gomock.InOrder(
			pool.EXPECT().Get().Times(1).Return(conn),
			conn.EXPECT().Do("SET", awsRequestId, body).Times(1).Return(nil, redisErr),
			conn.EXPECT().Close().Times(1).Return(nil),
		)

		// When
		res, err := app.Handler(ctx, &event)

		// Then
		if res != nil {
			t.Error("got non-nil, want nil")
		}

		if err == nil {
			t.Fatal("got nil, want error")
		}

		if err != redisErr {
			t.Errorf("got %s, want %s", err, redisErr)
		}
	})

	t.Run(`given a configured App
when Handler is called with an APIGatewayProxyRequest that has an populated Body
and there is an error publishing the message to the SNS topic
then the Body is stored in the configured Redis cache
and the response is nil
and there is an error`, func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pool := mock_repository.NewMockRedisPooler(ctrl)
		snsClient := mock_snsiface.NewMockSNSAPI(ctrl)
		snsTopicArn := "arn:aws:sns:us-east-1:000000000000:mytopic"

		app := New(pool, snsClient, snsTopicArn)

		awsRequestId := "943ad105-7543-11e6-a9ac-65e093327849"

		ctx := lambdacontext.NewContext(context.Background(), &lambdacontext.LambdaContext{
			AwsRequestID: awsRequestId,
		})

		body := "foo"

		event := events.APIGatewayProxyRequest{
			Body: body,
		}

		conn := mock_redis.NewMockConn(ctrl)

		snsError := errors.New("FUBAR")

		gomock.InOrder(
			pool.EXPECT().Get().Times(1).Return(conn),
			conn.EXPECT().Do("SET", awsRequestId, body).Times(1).Return("OK", nil),
			snsClient.EXPECT().Publish(&sns.PublishInput{
				Message:  &awsRequestId,
				TopicArn: &snsTopicArn,
			}).Times(1).Return(nil, snsError),
			conn.EXPECT().Close().Times(1).Return(nil),
		)

		// When
		res, err := app.Handler(ctx, &event)

		// Then
		if res != nil {
			t.Error("got non-nil, want nil")
		}

		if err == nil {
			t.Fatal("got nil, want non-nil")
		}

		if err != snsError {
			t.Errorf("got %s, want %s", err, snsError)
		}
	})

	t.Run(`given a configured App
when Handler is called with an APIGatewayProxyRequest that has an populated Body
and there is an error returning the Redis connection to the pool
then the Body is stored in the configured Redis cache
and the key the body was stored under is published to the configured SNS topic
and an APIGatewayProxyResponse is returned with a 200 status
and there is an error`, func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pool := mock_repository.NewMockRedisPooler(ctrl)
		snsClient := mock_snsiface.NewMockSNSAPI(ctrl)
		snsTopicArn := "arn:aws:sns:us-east-1:000000000000:mytopic"

		app := New(pool, snsClient, snsTopicArn)

		awsRequestId := "943ad105-7543-11e6-a9ac-65e093327849"

		ctx := lambdacontext.NewContext(context.Background(), &lambdacontext.LambdaContext{
			AwsRequestID: awsRequestId,
		})

		body := "foo"

		event := events.APIGatewayProxyRequest{
			Body: body,
		}

		conn := mock_redis.NewMockConn(ctrl)

		connError := errors.New("FUBAR")

		gomock.InOrder(
			pool.EXPECT().Get().Times(1).Return(conn),
			conn.EXPECT().Do("SET", awsRequestId, body).Times(1).Return("OK", nil),
			snsClient.EXPECT().Publish(&sns.PublishInput{
				Message:  &awsRequestId,
				TopicArn: &snsTopicArn,
			}).Times(1).Return(&sns.PublishOutput{}, nil),
			conn.EXPECT().Close().Times(1).Return(connError),
		)

		// When
		res, err := app.Handler(ctx, &event)

		// Then
		if res == nil {
			t.Fatal("got nil, want non-nil")
		}

		if res.StatusCode != http.StatusOK {
			t.Errorf("got %d, want %d", res.StatusCode, http.StatusOK)
		}

		if err == nil {
			t.Fatal("got nil, want non-nil")
		}

		if err != connError {
			t.Errorf("got %s, want %s", err, connError)
		}
	})
}

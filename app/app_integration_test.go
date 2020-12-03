// +build integration

package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Marchie/localstack-api-gateway-lambda-sns-example/pkg/repository"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/apigateway/apigatewayiface"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/gomodule/redigo/redis"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/plaid/go-envvar/envvar"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

type config struct {
	KubernetesPodIP                           string        `envvar:"KUBERNETES_POD_IP" default:""`
	TestEnabled                               bool          `envvar:"TEST_ENABLED" default:"false"`
	AWSEndpoint                               string        `envvar:"TEST_AWS_ENDPOINT"`
	RedisServerAddress                        string        `envvar:"TEST_REDIS_SERVER_ADDRESS"`
	TestTimeout                               time.Duration `envvar:"TEST_TIMEOUT" default:"30s"`
	SNSTopicConsumerEndpoint                  string        `envvar:"TEST_SNS_TOPIC_CONSUMER_ENDPOINT"`
	ServiceAvailabilityCheckInterval          time.Duration `envvar:"TEST_SERVICE_AVAILABILITY_CHECK_INTERVAL" default:"250ms"`
	APIGatewayAuthorizationType               string        `envvar:"TEST_API_GATEWAY_AUTHORIZATION_METHOD" default:"NONE"`
	APIGatewayHTTPMethods                     string        `envvar:"TEST_API_GATEWAY_HTTP_METHODS" default:"POST"`
	APIGatewayIntegrationPassThroughBehaviour string        `envvar:"TEST_API_GATEWAY_PASS_THROUGH_BEHAVIOUR" default:"WHEN_NO_MATCH"`
	APIGatewayIntegrationType                 string        `envvar:"TEST_API_GATEWAY_INTEGRATION_TYPE" default:"AWS_PROXY"`
	APIGatewayPathPart                        string        `envvar:"TEST_API_GATEWAY_PATH_PART" default:"test"`
	APIGatewayRESTAPIName                     string        `envvar:"TEST_API_GATEWAY_REST_API_NAME" default:"test"`
	APIGatewayStage                           string        `envvar:"TEST_API_GATEWAY_STAGE" default:"test"`
	LambdaFunctionCodePath                    string        `envvar:"TEST_LAMBDA_FUNCTION_CODE_PATH"`
	LambdaFunctionName                        string        `envvar:"TEST_LAMBDA_FUNCTION_NAME" default:"test"`
	LambdaHandler                             string        `envvar:"TEST_LAMBDA_HANDLER"`
	LambdaMemorySize                          int64         `envvar:"TEST_LAMBDA_MEMORY_SIZE" default:"128"`
	LambdaRole                                string        `envvar:"TEST_LAMBDA_ROLE" default:"arn:aws:iam::123456:role/irrelevant"`
	LambdaRuntime                             string        `envvar:"TEST_LAMBDA_RUNTIME" default:"go1.x"`
	LambdaTimeout                             int64         `envvar:"TEST_LAMBDA_TIMEOUT" default:"3"`
	SNSTopicName                              string        `envvar:"TEST_SNS_TOPIC_NAME" default:"request_bodies"`
}

type test struct {
	apiGatewayClient apigatewayiface.APIGatewayAPI
	lambdaClient     lambdaiface.LambdaAPI
	snsClient        snsiface.SNSAPI
	redisPool        repository.RedisPooler
	ctx              context.Context
	appConfig        *Config
	testConfig       *config
	functionCode     []byte
}

func newTest(ctx context.Context, functionCode []byte, appConfig *Config, testConfig *config, redisPool repository.RedisPooler, apiGatewayClient apigatewayiface.APIGatewayAPI, lambdaClient lambdaiface.LambdaAPI, snsClient snsiface.SNSAPI) *test {
	return &test{
		apiGatewayClient: apiGatewayClient,
		lambdaClient:     lambdaClient,
		snsClient:        snsClient,
		redisPool:        redisPool,
		ctx:              ctx,
		appConfig:        appConfig,
		testConfig:       testConfig,
		functionCode:     functionCode,
	}
}

func TestAppIntegration(t *testing.T) {
	t.Run(`given a configured stack
when a message is posted to the API Gateway endpoint
then the message body is stored in the configured Redis cache
and the key under which the message has been stored is published to an SNS topic`, func(t *testing.T) {
		wg := sync.WaitGroup{}
		chErr := make(chan error)

		go func() {
			defer func() {
				defer close(chErr)

				wg.Wait()
			}()

			// Given
			var testCfg config

			if err := envvar.Parse(&testCfg); err != nil {
				chErr <- err
				return
			}

			functionCode, err := ioutil.ReadFile(testCfg.LambdaFunctionCodePath)
			if err != nil {
				chErr <- err
				return
			}

			appConfig, err := NewConfig()
			if err != nil {
				chErr <- err
				return
			}

			if testCfg.KubernetesPodIP != "" {
				appConfig.AWSEndpoint = fmt.Sprintf("http://%s:%s", testCfg.KubernetesPodIP, strings.TrimLeft(appConfig.AWSEndpoint, ":"))
				appConfig.RedisServerAddress = fmt.Sprintf("%s:%s", testCfg.KubernetesPodIP, strings.TrimLeft(appConfig.RedisServerAddress, ":"))
			}

			sess, err := session.NewSession(&aws.Config{
				Endpoint: &testCfg.AWSEndpoint,
				Region:   &appConfig.AWSRegion,
			})
			if err != nil {
				chErr <- err
				return
			}

			apiGatewayClient := apigateway.New(sess)

			lambdaClient := lambda.New(sess)

			snsClient := sns.New(sess)

			redisPool := &redis.Pool{
				Dial: func() (redis.Conn, error) {
					return redis.Dial("tcp", testCfg.RedisServerAddress)
				},
			}
			defer func() {
				if err := redisPool.Close(); err != nil {
					chErr <- err
				}
			}()

			httpClient := http.Client{
				Timeout: testCfg.TestTimeout - time.Millisecond*500,
			}

			ctx, cancelFunc := context.WithTimeout(context.Background(), testCfg.TestTimeout)

			if err := checkAllServicesAreAvailable(ctx, testCfg.ServiceAvailabilityCheckInterval, redisPool, apiGatewayClient, lambdaClient, snsClient); err != nil {
				chErr <- err
				return
			}

			tester := newTest(ctx, functionCode, appConfig, &testCfg, redisPool, apiGatewayClient, lambdaClient, snsClient)

			chSubscriptionConfirmed := make(chan struct{})
			chSNSNotification := make(chan snsNotification)

			chConsumerUrl, chSNSTopicConsumerErr := tester.setupSNSTopicConsumer(createSubscriptionConfirmationHandler(snsClient, chSubscriptionConfirmed), createNotificationHandler(chSNSNotification))
			wg.Add(1)
			go handleErrorChannel(&wg, chErr, chSNSTopicConsumerErr)

			var consumerUrl *url.URL

			select {
			case consumerUrl = <-chConsumerUrl:
			case <-ctx.Done():
				return
			}

			chApiEndpoint, chStackErr := tester.setUpStack(consumerUrl)
			wg.Add(1)
			go handleErrorChannel(&wg, chErr, chStackErr)

			var apiEndpoint *url.URL

			select {
			case apiEndpoint = <-chApiEndpoint:
			case <-ctx.Done():
				return
			}

			// When
			body := "foo"

			req, err := http.NewRequest("POST", apiEndpoint.String(), bytes.NewBufferString(body))
			if err != nil {
				t.Fatal(err)
			}

			resp, err := httpClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}

			// Then
			if resp.StatusCode != http.StatusOK {
				t.Errorf("got %d, want %d", resp.StatusCode, http.StatusOK)
			}

			conn := redisPool.Get()
			defer func() {
				if connErr := conn.Close(); connErr != nil {
					if err == nil {
						err = connErr
					}
				}
			}()

			select {
			case msg := <-chSNSNotification:
				key := msg.Message

				res, err := redis.String(conn.Do("GET", key))
				if err != nil {
					t.Error(err)
				}

				if res != body {
					t.Errorf("got %s, want %s", res, body)
				}

				cancelFunc()
				return
			case <-ctx.Done():
				t.Error(ctx.Err())
				return
			}
		}()

		for err := range chErr {
			t.Error(err)
		}
	})
}

func (s *test) setupSNSTopicConsumer(subscriptionConfirmationHandler, notificationHandler func(body io.ReadCloser) error) (<-chan *url.URL, <-chan error) {
	chSNSTopicConsumerURL := make(chan *url.URL)
	chErr := make(chan error)

	go func() {
		defer close(chErr)
		snsTopicConsumerUrl, err := url.Parse(s.testConfig.SNSTopicConsumerEndpoint)
		if err != nil {
			chErr <- err
			return
		}

		hostAndPort := strings.Split(snsTopicConsumerUrl.Host, ":")
		if len(hostAndPort) != 2 {
			chErr <- errors.New("expected SNSTopicConsumerEndpoint to be defined as http://<host>:<port>")
			return
		}

		port := hostAndPort[1]

		server := &http.Server{
			Addr: fmt.Sprintf(":%s", port),
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Header.Get("x-amz-sns-message-type") {
				case "SubscriptionConfirmation":
					if err := subscriptionConfirmationHandler(r.Body); err != nil {
						chErr <- err
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
				case "Notification":
					if err := notificationHandler(r.Body); err != nil {
						chErr <- err
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
				case "UnsubscribeConfirmation":
				default:
					chErr <- fmt.Errorf("unrecognised SNS message type '%s'", r.Header.Get("x-amz-sns-message-type"))
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				w.WriteHeader(http.StatusOK)
			}),
		}

		go func() {
			if err := server.ListenAndServe(); err != http.ErrServerClosed {
				chErr <- errors.Wrap(err, "HTTP server for consuming SNS topic failed")
			}
		}()
		defer func() {
			if err := server.Shutdown(s.ctx); err != nil {
				if err.Error() != "context canceled" {
					chErr <- errors.Wrap(err, "HTTP server for consuming SNS topic did not shut down gracefully")
				}
			}
		}()

		go func() {
			defer close(chSNSTopicConsumerURL)

			chSNSTopicConsumerURL <- snsTopicConsumerUrl
		}()

		// Block until the tests have completed
		<-s.ctx.Done()
	}()

	return chSNSTopicConsumerURL, chErr
}

func createSubscriptionConfirmationHandler(snsClient snsiface.SNSAPI, chSubscriptionConfirmed chan struct{}) func(body io.ReadCloser) error {
	return func(body io.ReadCloser) error {
		var msg snsSubscriptionConfirmation

		if err := json.NewDecoder(body).Decode(&msg); err != nil {
			return err
		}

		if err := body.Close(); err != nil {
			return err
		}

		if err := ConfirmSubscription(snsClient, msg.Token, msg.TopicArn); err != nil {
			return err
		}

		defer close(chSubscriptionConfirmed)

		return nil
	}
}

func createNotificationHandler(chNotificationMessages chan snsNotification) func(body io.ReadCloser) error {
	return func(body io.ReadCloser) error {
		var msg snsNotification

		if err := json.NewDecoder(body).Decode(&msg); err != nil {
			return err
		}

		if err := body.Close(); err != nil {
			return err
		}

		chNotificationMessages <- msg

		return nil
	}
}

func (s *test) handleSubscriptionConfirmation(body io.ReadCloser) error {
	var msg snsSubscriptionConfirmation

	if err := json.NewDecoder(body).Decode(&msg); err != nil {
		return err
	}
	if err := body.Close(); err != nil {
		return err
	}

	if err := ConfirmSubscription(s.snsClient, msg.Token, msg.TopicArn); err != nil {
		return err
	}

	return nil
}

func checkAllServicesAreAvailable(ctx context.Context, checkInterval time.Duration, redisPool repository.RedisPooler, apiGatewayClient apigatewayiface.APIGatewayAPI, lambdaClient lambdaiface.LambdaAPI, snsClient snsiface.SNSAPI) error {
	wg := sync.WaitGroup{}
	chErr := make(chan error)

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := RedisServerIsAvailable(ctx, redisPool, checkInterval); err != nil {
			chErr <- err
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := APIGatewayServiceIsAvailable(ctx, apiGatewayClient, checkInterval); err != nil {
			chErr <- err
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := LambdaServiceIsAvailable(ctx, lambdaClient, checkInterval); err != nil {
			chErr <- err
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := SNSServiceIsAvailable(ctx, snsClient, checkInterval); err != nil {
			chErr <- err
		}
	}()

	go func() {
		defer close(chErr)

		wg.Wait()
	}()

	var result *multierror.Error

	for err := range chErr {
		result = multierror.Append(result, err)
	}

	return result.ErrorOrNil()
}

func (s *test) setUpStack(consumerUrl *url.URL) (<-chan *url.URL, <-chan error) {
	chApiEndpoint := make(chan *url.URL)
	chErr := make(chan error)

	go func() {
		defer close(chErr)

		topicArn, err := CreateSNSTopic(s.snsClient, s.testConfig.SNSTopicName)
		if err != nil {
			chErr <- err
			return
		}
		defer func() {
			if err := DeleteSNSTopic(s.snsClient, topicArn); err != nil {
				chErr <- err
			}
		}()

		// Update appConfig with topicArn
		s.appConfig.SNSTopicARN = topicArn

		lambdaArn, err := CreateLambdaFunction(s.lambdaClient, s.functionCode, s.testConfig.LambdaFunctionName, s.testConfig.LambdaHandler, s.testConfig.LambdaRole, s.testConfig.LambdaRuntime, s.testConfig.LambdaMemorySize, s.testConfig.LambdaTimeout, appConfigToLambdaEnvironmentMap(s.appConfig))
		if err != nil {
			chErr <- err
			return
		}
		defer func() {
			if err := DeleteLambdaFunction(s.lambdaClient, lambdaArn); err != nil {
				chErr <- err
			}
		}()

		restApiId, err := CreateAPIGatewayRestAPI(s.apiGatewayClient, s.testConfig.APIGatewayRESTAPIName)
		if err != nil {
			chErr <- err
			return
		}
		defer func() {
			if err := DeleteAPIGatewayRestAPI(s.apiGatewayClient, restApiId); err != nil {
				chErr <- err
			}
		}()

		parentResourceId, err := FindAPIGatewayParentResourceId(s.apiGatewayClient, restApiId)
		if err != nil {
			chErr <- err
			return
		}

		resourceId, err := CreateAPIGatewayResource(s.apiGatewayClient, parentResourceId, s.testConfig.APIGatewayPathPart, restApiId)
		if err != nil {
			chErr <- err
			return
		}
		defer func() {
			if err := DeleteAPIGatewayResource(s.apiGatewayClient, resourceId, restApiId); err != nil {
				chErr <- err
			}
		}()

		httpMethods := strings.Split(s.testConfig.APIGatewayHTTPMethods, ",")
		for i := range httpMethods {
			httpMethod := httpMethods[i]
			if err := PutAPIGatewayMethod(s.apiGatewayClient, httpMethod, resourceId, restApiId, s.testConfig.APIGatewayAuthorizationType); err != nil {
				chErr <- err
				return
			}
			defer func() {
				if err := DeleteAPIGatewayMethod(s.apiGatewayClient, httpMethod, resourceId, restApiId); err != nil {
					chErr <- err
				}
			}()

			if err := PutAPIGatewayIntegration(s.apiGatewayClient, lambdaArn, httpMethod, resourceId, restApiId, s.testConfig.APIGatewayIntegrationType, s.testConfig.APIGatewayIntegrationPassThroughBehaviour, GetLambdaURI(s.appConfig.AWSRegion, lambdaArn)); err != nil {
				chErr <- err
				return
			}
			defer func() {
				if err := DeleteAPIGatewayIntegration(s.apiGatewayClient, httpMethod, resourceId, restApiId); err != nil {
					chErr <- err
				}
			}()
		}

		deploymentId, err := CreateAPIGatewayDeployment(s.apiGatewayClient, restApiId, s.testConfig.APIGatewayStage)
		if err != nil {
			chErr <- err
			return
		}
		defer func() {
			if err := DeleteAPIGatewayDeployment(s.apiGatewayClient, deploymentId, restApiId); err != nil {
				chErr <- err
			}
		}()

		go func() {
			defer close(chApiEndpoint)

			apiEndpoint, err := url.Parse(fmt.Sprintf("%s/restapis/%s/%s/_user_request_/%s", s.testConfig.AWSEndpoint, restApiId, s.testConfig.APIGatewayStage, s.testConfig.APIGatewayPathPart))
			if err != nil {
				chErr <- err
				return
			}

			chApiEndpoint <- apiEndpoint
		}()

		subscriptionArn, err := SubscribeToSNSTopic(s.snsClient, consumerUrl.String(), consumerUrl.Scheme, topicArn)
		if err != nil {
			chErr <- err
			return
		}
		defer func() {
			if err := UnsubscribeFromSNSTopic(s.snsClient, subscriptionArn); err != nil {
				chErr <- err
			}
		}()

		<-s.ctx.Done()
	}()

	return chApiEndpoint, chErr
}

func appConfigToLambdaEnvironmentMap(appConfig *Config) map[string]*string {
	return map[string]*string{
		"AWS_ENDPOINT":          &appConfig.AWSEndpoint,
		"AWS_REGION":            &appConfig.AWSRegion,
		"AWS_ACCESS_KEY_ID":     &appConfig.AWSAccessKeyID,
		"AWS_SECRET_ACCESS_KEY": &appConfig.AWSSecretAccessKey,
		"REDIS_SERVER_ADDRESS":  &appConfig.RedisServerAddress,
		"SNS_TOPIC_ARN":         &appConfig.SNSTopicARN,
	}
}

func handleErrorChannel(wg *sync.WaitGroup, chErrMain chan<- error, chErrSource <-chan error) {
	defer wg.Done()

	for err := range chErrSource {
		chErrMain <- err
	}
}

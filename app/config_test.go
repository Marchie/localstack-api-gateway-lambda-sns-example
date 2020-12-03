package app

import (
	"github.com/alicebob/miniredis/v2"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/gomodule/redigo/redis"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// setEnv is a test helper to set up the environment for a test.
// It returns a function that should be called to reset the environment back to its original state
func setEnv(t *testing.T, envMap map[string]*string) (resetEnvFunc func()) {
	t.Helper()

	originalValues := make(map[string]*string)

	for key, valueToSet := range envMap {
		originalValue, exists := os.LookupEnv(key)

		if exists {
			originalValues[key] = &originalValue
		} else {
			originalValues[key] = nil
		}

		if valueToSet != nil {
			if err := os.Setenv(key, *valueToSet); err != nil {
				t.Error(err)
			}
		} else {
			if err := os.Unsetenv(key); err != nil {
				t.Error(err)
			}
		}

	}

	return func() {
		for key, originalValue := range originalValues {
			if originalValue != nil {
				if err := os.Setenv(key, *originalValue); err != nil {
					t.Error(err)
				}
				continue
			}

			if err := os.Unsetenv(key); err != nil {
				t.Error(err)
			}
		}
	}
}

func TestNewConfig(t *testing.T) {
	t.Run(`given the environment is not configured
when NewConfig() is called
then an error is returned`, func(t *testing.T) {
		// Given
		envMap := map[string]*string{
			"AWS_REGION":            nil,
			"AWS_ACCESS_KEY_ID":     nil,
			"AWS_SECRET_ACCESS_KEY": nil,
			"REDIS_SERVER_ADDRESS":  nil,
			"SNS_TOPIC_ARN":         nil,
		}

		resetEnvFunc := setEnv(t, envMap)
		defer resetEnvFunc()

		// When
		_, err := NewConfig()

		// Then
		if err == nil {
			t.Fatal("got nil, want error")
		}

		expErrMsg :=
			`envvar: Missing required environment variable: AWS_REGION
envvar: Missing required environment variable: AWS_ACCESS_KEY_ID
envvar: Missing required environment variable: AWS_SECRET_ACCESS_KEY
envvar: Missing required environment variable: REDIS_SERVER_ADDRESS
envvar: Missing required environment variable: SNS_TOPIC_ARN`

		if err.Error() != expErrMsg {
			t.Errorf("got:\n%s\nwant:\n%s", err, expErrMsg)
		}
	})

	t.Run(`given the environment is configured
when NewConfig() is called
then a Config is returned`, func(t *testing.T) {
		// Given
		awsRegion := "us-east-1"
		awsAccessKeyId := "test"
		awsSecretAccessKey := "test"
		redisServerAddress := "localhost:6379"
		snsTopicArn := "arn:aws:sns:us-east-1:000000000000:mytopic"

		envMap := map[string]*string{
			"AWS_REGION":            &awsRegion,
			"AWS_ACCESS_KEY_ID":     &awsAccessKeyId,
			"AWS_SECRET_ACCESS_KEY": &awsSecretAccessKey,
			"REDIS_SERVER_ADDRESS":  &redisServerAddress,
			"SNS_TOPIC_ARN":         &snsTopicArn,
		}

		resetEnvFunc := setEnv(t, envMap)
		defer resetEnvFunc()

		// When
		cfg, err := NewConfig()

		// Then
		if err != nil {
			t.Fatal(err)
		}

		if cfg.RedisServerAddress != redisServerAddress {
			t.Errorf("got:  %s\nwant: %s", cfg.RedisServerAddress, redisServerAddress)
		}

		if cfg.SNSTopicARN != snsTopicArn {
			t.Errorf("got:  %s\nwant: %s", cfg.SNSTopicARN, snsTopicArn)
		}
	})
}

func TestConfig_NewRedisPool(t *testing.T) {
	t.Run(`given a valid Config
when NewRedisPool is called
then a configured redis.Pool is returned`, func(t *testing.T) {
		// Given
		s, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		awsRegion := "us-east-1"
		awsAccessKeyId := "test"
		awsSecretAccessKey := "test"
		redisServerAddress := s.Addr()
		snsTopicArn := "arn:aws:sns:us-east-1:000000000000:mytopic"

		envMap := map[string]*string{
			"AWS_REGION":            &awsRegion,
			"AWS_ACCESS_KEY_ID":     &awsAccessKeyId,
			"AWS_SECRET_ACCESS_KEY": &awsSecretAccessKey,
			"REDIS_SERVER_ADDRESS":  &redisServerAddress,
			"SNS_TOPIC_ARN":         &snsTopicArn,
		}

		resetEnvFunc := setEnv(t, envMap)
		defer resetEnvFunc()

		cfg, err := NewConfig()
		if err != nil {
			t.Fatal(err)
		}

		// When
		pool := cfg.NewRedisPool()
		defer func() {
			if err := pool.Close(); err != nil {
				t.Error(err)
			}
		}()

		// Then
		conn := pool.Get()

		resp, err := redis.String(conn.Do("PING"))
		if err != nil {
			t.Error(err)
		}

		expResp := "PONG"
		if resp != expResp {
			t.Errorf("got %s, want %s", resp, expResp)
		}
	})
}

func TestConfig_NewSNSClient(t *testing.T) {
	t.Run(`given a valid Config
when NewSNSClient is called
then a configured SNS client is returned`, func(t *testing.T) {
		// Given
		callMade := make(chan struct{})

		fakeSns := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer close(callMade)

			w.WriteHeader(200)
		}))
		defer fakeSns.Close()

		awsEndpoint := fakeSns.URL
		awsRegion := "us-east-1"
		awsAccessKeyId := "test"
		awsSecretAccessKey := "test"
		redisServerAddress := "localhost:6379"
		snsTopicArn := "arn:aws:sns:us-east-1:000000000000:mytopic"

		envMap := map[string]*string{
			"AWS_ENDPOINT":          &awsEndpoint,
			"AWS_REGION":            &awsRegion,
			"AWS_ACCESS_KEY_ID":     &awsAccessKeyId,
			"AWS_SECRET_ACCESS_KEY": &awsSecretAccessKey,
			"REDIS_SERVER_ADDRESS":  &redisServerAddress,
			"SNS_TOPIC_ARN":         &snsTopicArn,
		}

		resetEnvFunc := setEnv(t, envMap)
		defer resetEnvFunc()

		cfg, err := NewConfig()
		if err != nil {
			t.Fatal(err)
		}

		// When
		snsClient, err := cfg.NewSNSClient()
		if err != nil {
			t.Fatal(err)
		}

		// Then
		if _, err := snsClient.ListTopics(&sns.ListTopicsInput{}); err != nil {
			t.Fatal(err)
		}

		testTimeout := time.Second

		select {
		case <-time.After(testTimeout):
			t.Errorf("test timed out after %s", testTimeout)
		case <-callMade:
		}
	})
}

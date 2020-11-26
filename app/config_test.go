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

// resetEnv is a test helper to reset an environment variable to its previous state.
func resetEnv(t *testing.T, key string, exists bool, originalValue string) {
	t.Helper()

	if exists {
		if err := os.Setenv(key, originalValue); err != nil {
			t.Error(err)
		}

		return
	}

	if err := os.Unsetenv(key); err != nil {
		t.Error(err)
	}
}

func TestNewConfig(t *testing.T) {
	t.Run(`given the environment is not configured
when NewConfig() is called
then an error is returned`, func(t *testing.T) {
		// Given
		// No configuration

		// When
		_, err := NewConfig()

		// Then
		if err == nil {
			t.Fatal("got nil, want error")
		}

		expErrMsg :=
			`envvar: Missing required environment variable: REDIS_SERVER_ADDRESS
envvar: Missing required environment variable: SNS_TOPIC_ARN`

		if err.Error() != expErrMsg {
			t.Errorf("got:\n%s\nwant:\n%s", err, expErrMsg)
		}
	})

	t.Run(`given the environment is configured
when NewConfig() is called
then a Config is returned`, func(t *testing.T) {
		// Given
		redisServerAddressKey := "REDIS_SERVER_ADDRESS"
		originalValue, exists := os.LookupEnv(redisServerAddressKey)
		defer resetEnv(t, redisServerAddressKey, exists, originalValue)

		redisServerAddress := "localhost:6379"
		if err := os.Setenv(redisServerAddressKey, redisServerAddress); err != nil {
			t.Fatal(err)
		}

		snsTopicArnKey := "SNS_TOPIC_ARN"
		originalValue, exists = os.LookupEnv(snsTopicArnKey)
		defer resetEnv(t, snsTopicArnKey, exists, originalValue)

		snsTopicArn := "arn:aws:sns:us-east-1:000000000000:mytopic"
		if err := os.Setenv(snsTopicArnKey, snsTopicArn); err != nil {
			t.Fatal(err)
		}

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

		redisServerAddressKey := "REDIS_SERVER_ADDRESS"
		originalValue, exists := os.LookupEnv(redisServerAddressKey)
		defer resetEnv(t, redisServerAddressKey, exists, originalValue)

		redisServerAddress := s.Addr()
		if err := os.Setenv(redisServerAddressKey, redisServerAddress); err != nil {
			t.Fatal(err)
		}

		snsTopicArnKey := "SNS_TOPIC_ARN"
		originalValue, exists = os.LookupEnv(snsTopicArnKey)
		defer resetEnv(t, snsTopicArnKey, exists, originalValue)

		snsTopicArn := "arn:aws:sns:us-east-1:000000000000:mytopic"
		if err := os.Setenv(snsTopicArnKey, snsTopicArn); err != nil {
			t.Fatal(err)
		}

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

		redisServerAddressKey := "REDIS_SERVER_ADDRESS"
		originalValue, exists := os.LookupEnv(redisServerAddressKey)
		defer resetEnv(t, redisServerAddressKey, exists, originalValue)

		redisServerAddress := "localhost:6379"
		if err := os.Setenv(redisServerAddressKey, redisServerAddress); err != nil {
			t.Fatal(err)
		}

		awsEndpointKey := "AWS_ENDPOINT"
		originalValue, exists = os.LookupEnv(awsEndpointKey)
		defer resetEnv(t, awsEndpointKey, exists, originalValue)

		awsEndpoint := fakeSns.URL
		if err := os.Setenv(awsEndpointKey, awsEndpoint); err != nil {
			t.Fatal(err)
		}

		awsRegionKey := "AWS_REGION"
		originalValue, exists = os.LookupEnv(awsRegionKey)
		defer resetEnv(t, awsRegionKey, exists, originalValue)

		awsRegion := "us-east-1"
		if err := os.Setenv(awsRegionKey, awsRegion); err != nil {
			t.Fatal(err)
		}

		snsTopicArnKey := "SNS_TOPIC_ARN"
		originalValue, exists = os.LookupEnv(snsTopicArnKey)
		defer resetEnv(t, snsTopicArnKey, exists, originalValue)

		snsTopicArn := "arn:aws:sns:us-east-1:000000000000:mytopic"
		if err := os.Setenv(snsTopicArnKey, snsTopicArn); err != nil {
			t.Fatal(err)
		}

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

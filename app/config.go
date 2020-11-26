package app

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/gomodule/redigo/redis"
	"github.com/plaid/go-envvar/envvar"
)

type Config struct {
	AWSEndpoint        string `envvar:"AWS_ENDPOINT" default:""`
	RedisServerAddress string `envvar:"REDIS_SERVER_ADDRESS"`
	SNSTopicARN        string `envvar:"SNS_TOPIC_ARN"`
}

func NewConfig() (*Config, error) {
	var cfg Config

	if err := envvar.Parse(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) NewRedisPool() *redis.Pool {
	return &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", c.RedisServerAddress)
		},
	}
}

func (c *Config) NewSNSClient() (snsiface.SNSAPI, error) {
	sess, err := c.newAWSSession()
	if err != nil {
		return nil, err
	}

	return sns.New(sess), nil
}

func (c *Config) newAWSSession() (*session.Session, error) {
	return session.NewSession(&aws.Config{
		Endpoint: &c.AWSEndpoint,
	})
}

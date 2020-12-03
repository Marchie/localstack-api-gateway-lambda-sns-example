package main

import (
	"github.com/Marchie/localstack-api-gateway-lambda-sns-example/app"
	"github.com/aws/aws-lambda-go/lambda"
	"log"
)

func main() {
	cfg, err := app.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	pool := cfg.NewRedisPool()

	snsClient, err := cfg.NewSNSClient()
	if err != nil {
		log.Fatal(err)
	}

	a := app.New(pool, snsClient, cfg.SNSTopicARN)

	lambda.Start(a.Handler)
}

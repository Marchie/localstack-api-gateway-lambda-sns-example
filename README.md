# localstack-api-gateway-lambda-sns-example

This example demonstrates how we can test an AWS stack on our local machine using [LocalStack](https://github.com/localstack/localstack).

The specification for this Lambda function is as follows:

- It is triggered by an AWS API Gateway Proxy Request event;
- It stores the body from the request in a configured Redis cache, using the AWS Request ID as the key; and
- It publishes the key for the cached record to a configured SNS topic.

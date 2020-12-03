// +build integration

package app

import (
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/pkg/errors"
	"time"
)

type snsSubscriptionConfirmation struct {
	Type             string
	MessageId        string
	Token            string
	TopicArn         string
	Message          string
	SubscribeURL     string
	Timestamp        time.Time
	SignatureVersion string
	Signature        string
	SigningCertURL   string
}

type snsNotification struct {
	Type             string
	MessageId        string
	TopicArn         string
	Subject          string
	Message          string
	Timestamp        time.Time
	SignatureVersion string
	Signature        string
	SigningCertURL   string
	UnsubscribeURL   string
}

type snsUnsubscribeConfirmation struct {
	Type             string
	MessageId        string
	Token            string
	TopicArn         string
	Message          string
	SubscribeURL     string
	Timestamp        time.Time
	SignatureVersion string
	Signature        string
	SigningCertURL   string
}

func CreateSNSTopic(snsClient snsiface.SNSAPI, name string) (string, error) {
	createTopicOutput, err := snsClient.CreateTopic(&sns.CreateTopicInput{
		Name: &name,
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed creating SNS topic '%s'", name)
	}

	return *createTopicOutput.TopicArn, nil
}

func DeleteSNSTopic(snsClient snsiface.SNSAPI, topicArn string) error {
	if _, err := snsClient.DeleteTopic(&sns.DeleteTopicInput{
		TopicArn: &topicArn,
	}); err != nil {
		return errors.Wrapf(err, "failed deleting SNS topic '%s'", topicArn)
	}

	return nil
}

func ConfirmSubscription(snsClient snsiface.SNSAPI, token, topicArn string) error {
	if _, err := snsClient.ConfirmSubscription(&sns.ConfirmSubscriptionInput{
		Token:    &token,
		TopicArn: &topicArn,
	}); err != nil {
		return errors.Wrapf(err, "failed to confirm subscription for topic '%s' with token '%s'", topicArn, token)
	}

	return nil
}

func SubscribeToSNSTopic(snsClient snsiface.SNSAPI, endpoint, protocol, topicArn string) (string, error) {
	subscribeOutput, err := snsClient.Subscribe(&sns.SubscribeInput{
		Endpoint: &endpoint,
		Protocol: &protocol,
		TopicArn: &topicArn,
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed to subscribe to endpoint '%s' with protocol '%s' to SNS topic '%s'", endpoint, protocol, topicArn)
	}

	return *subscribeOutput.SubscriptionArn, nil
}

func UnsubscribeFromSNSTopic(snsClient snsiface.SNSAPI, subscriptionArn string) error {
	if _, err := snsClient.Unsubscribe(&sns.UnsubscribeInput{
		SubscriptionArn: &subscriptionArn,
	}); err != nil {
		return errors.Wrapf(err, "failed to unsubscribe '%s'", subscriptionArn)
	}

	return nil
}

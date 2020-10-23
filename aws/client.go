package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
)

type Client struct {
	Region           string
	stackTemplateRUL string
	Sess             *session.Session
}

func NewAWSClient(region string, stackTemplateRUL string, sp credentials.StaticProvider) (*Client, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewCredentials(&sp),
	})
	if err != nil {
		return nil, err
	}

	_, err = sess.Config.Credentials.Get()
	if err != nil {
		return nil, err
	}

	return &Client{stackTemplateRUL: stackTemplateRUL, Region: region, Sess: sess}, nil
}

func NewDefAWSClient() (*Client, error) {
	region := "us-east-2"
	stackTemplateRUL := ""

	return NewAWSClient(region, stackTemplateRUL, credentials.StaticProvider{
		Value: credentials.Value{
			AccessKeyID:     "",
			SecretAccessKey: "",
		},
	})
}

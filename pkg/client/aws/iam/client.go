package iam

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

type Client struct {
	svc *iam.IAM
}

func New() (*Client, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	svc := iam.New(sess)

	return &Client{svc}, nil
}

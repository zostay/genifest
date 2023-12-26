package iam

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
)

const MaxAccessKeys = 2

var (
	ErrNoOldKey     = fmt.Errorf("no old key")
	ErrNoKey        = fmt.Errorf("no key")
	ErrRecentlyUsed = fmt.Errorf("recently used")
)

func (c *Client) BestAccessKeyForUser(user string) (string, time.Time, error) {
	_, newKey, err := c.GetAccessKeys(user)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("c.GetAccessKeys(): %w", err)
	}

	if newKey == nil {
		return "", time.Time{}, nil
	}

	return aws.StringValue(newKey.AccessKeyId), aws.TimeValue(newKey.CreateDate), nil
}

func (c *Client) GetAccessKeys(user string) (*iam.AccessKeyMetadata, *iam.AccessKeyMetadata, error) {
	ak, err := c.svc.ListAccessKeys(&iam.ListAccessKeysInput{
		UserName: aws.String(user),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("c.svc.ListAccessKeys(%q): %w", user, err)
	}

	oldKey, newKey := examineKeys(ak.AccessKeyMetadata)
	return oldKey, newKey, nil
}

func examineKeys(akmds []*iam.AccessKeyMetadata) (*iam.AccessKeyMetadata, *iam.AccessKeyMetadata) {
	var (
		oldestTime = time.Now()
		oldestKey  *iam.AccessKeyMetadata
		newestTime time.Time
		newestKey  *iam.AccessKeyMetadata
	)
	for _, akmd := range akmds {
		if akmd.CreateDate != nil && akmd.CreateDate.Before(oldestTime) {
			oldestTime = *akmd.CreateDate
			oldestKey = akmd
		}
		if akmd.CreateDate != nil && akmd.CreateDate.After(newestTime) {
			newestTime = *akmd.CreateDate
			newestKey = akmd
		}
	}

	return oldestKey, newestKey
}

func (c *Client) RotateAccessKeyForUser(user string) (string, string, error) {
	oldKey, _, err := c.GetAccessKeys(user)
	if err != nil {
		return "", "", fmt.Errorf("c.GetAccessKeys(): %w", err)
	}

	if oldKey != nil {
		_, err := c.svc.DeleteAccessKey(&iam.DeleteAccessKeyInput{
			UserName:    aws.String(user),
			AccessKeyId: oldKey.AccessKeyId,
		})
		if err != nil {
			return "", "", fmt.Errorf("c.svc.DeleteAccessKey(): %w", err)
		}
	}

	ck, err := c.svc.CreateAccessKey(&iam.CreateAccessKeyInput{
		UserName: aws.String(user),
	})
	if err != nil {
		return "", "", fmt.Errorf("c.svc.CreateAccessKey(): %w", err)
	}

	accessKey := aws.StringValue(ck.AccessKey.AccessKeyId)
	secretKey := aws.StringValue(ck.AccessKey.SecretAccessKey)

	return accessKey, secretKey, nil
}

func (c *Client) RotateAccessKeyForUserFollowup(
	user string,
	recentUse time.Duration,
) error {
	oldKey, _, err := c.GetAccessKeys(user)
	if err != nil {
		return fmt.Errorf("c.GetAccessKeys(): %w", err)
	}

	if oldKey == nil {
		return ErrNoOldKey
	}

	lu, err := c.svc.GetAccessKeyLastUsed(&iam.GetAccessKeyLastUsedInput{
		AccessKeyId: oldKey.AccessKeyId,
	})
	if err != nil {
		return fmt.Errorf("c.svc.GetAccessKeyLastUser(): %w", err)
	}

	lud := *lu.AccessKeyLastUsed.LastUsedDate
	if time.Since(lud) <= recentUse {
		return ErrRecentlyUsed
	}

	_, err = c.svc.UpdateAccessKey(&iam.UpdateAccessKeyInput{
		UserName:    aws.String(user),
		AccessKeyId: oldKey.AccessKeyId,
		Status:      aws.String("Inactive"),
	})

	if err != nil {
		return fmt.Errorf("c.svc.UpdateAccessKey(): %w", err)
	}

	return nil
}

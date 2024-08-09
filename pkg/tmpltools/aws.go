package tmpltools

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/efs"
)

type AWS struct {
	Region string
}

// DDBLookup returns a function that performs a simple map lookup function in
// DynamoDB.
func (a AWS) DDBLookup(table, field string, key map[string]any) (string, error) {
	ddbKey := make(map[string]*dynamodb.AttributeValue, len(key))
	for k, v := range key {
		ddbKey[k] = &dynamodb.AttributeValue{S: aws.String(v.(string))}
	}
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(a.Region),
	})
	if err != nil {
		return "", err
	}
	ddbc := dynamodb.New(sess)
	in := dynamodb.GetItemInput{
		TableName: aws.String(table),
		Key:       ddbKey,
	}
	out, err := ddbc.GetItem(&in)
	if err != nil {
		return "", err
	}

	fps := strings.SplitN(field, ".", 2)
	fieldName, fieldType := fps[0], fps[1]

	if out.Item == nil {
		return "", fmt.Errorf("no counter named %s", key["Project"])
	}

	switch fieldType {
	case "S":
		return aws.StringValue(out.Item[fieldName].S), nil
	case "N":
		return aws.StringValue(out.Item[fieldName].N), nil
	default:
		return "", fmt.Errorf("unknown field type %q", fieldType)
	}
}

// DescribeEfsFileSystemId returns a function that lookups up an EFS file
// systems description.
func (a AWS) DescribeEfsFileSystemId(token string) (string, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(a.Region),
	})
	if err != nil {
		return "", err
	}
	efsc := efs.New(sess)
	in := efs.DescribeFileSystemsInput{
		CreationToken: aws.String(token),
	}
	out, err := efsc.DescribeFileSystems(&in)
	if err != nil {
		return "", err
	}
	return aws.StringValue(out.FileSystems[0].FileSystemId), nil
}

// DescribeEfsMountTargets Lookup EFS mount targets.
func (a AWS) DescribeEfsMountTargets(id string) (*efs.DescribeMountTargetsOutput, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(a.Region),
	})
	if err != nil {
		return nil, err
	}
	efsc := efs.New(sess)
	in := efs.DescribeMountTargetsInput{
		FileSystemId: aws.String(id),
	}
	out, err := efsc.DescribeMountTargets(&in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

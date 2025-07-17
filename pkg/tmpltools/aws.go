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

var scalarTypes = map[string]struct{}{
	"N":    {},
	"S":    {},
	"BOOL": {},
}

func lookupAttributeValue(field string, val map[string]*dynamodb.AttributeValue) (string, error) {
	parts := strings.SplitN(field, ".", 3)
	if len(parts) < 2 {
		return "", fmt.Errorf("%s is not a valid dynamodb attribute name must be of a form similar to name.S or name.M.name.N", field)
	}

	var next string
	name, fieldType := parts[0], parts[1]
	if len(parts) == 3 {
		next = parts[2]
	}

	if _, isScalar := scalarTypes[fieldType]; isScalar && next != "" {
		return "", fmt.Errorf("field expression %q has a nested value that does not make sense", field)
	}

	nextVal := val[name]
	switch fieldType {
	case "N":
		return aws.StringValue(nextVal.N), nil
	case "S":
		return aws.StringValue(nextVal.S), nil
	case "BOOL":
		bv := aws.BoolValue(nextVal.BOOL)
		return fmt.Sprintf("%t", bv), nil
	case "M":
		return lookupAttributeValue(next, nextVal.M)
	default:
		return "", fmt.Errorf("field expression %q has an unsupported value type (only N, S, BOOL, and M are supported)", field)
	}
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

	return lookupAttributeValue(field, out.Item)
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

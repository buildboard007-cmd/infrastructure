package data

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	ssmRepository SSMRepository
)

func String(v string) *string {
	return &v
}

type MockSSMClient struct {
	TestSuccess bool
}

func InitializeSSMClient(testSuccess bool) SSMRepository {
	mock := &MockSSMClient{
		TestSuccess: testSuccess,
	}

	return &SSMDao{
		SSM:    mock,
		Logger: logrus.New(),
	}
}

func (m *MockSSMClient) GetParametersByPath(ctx context.Context, input *ssm.GetParametersByPathInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error) {
	if m.TestSuccess {
		result := &ssm.GetParametersByPathOutput{
			Parameters: []types.Parameter{
				{
					Name:  String("param1"),
					Value: String("value1"),
				},
				{
					Name:  String("param2"),
					Value: String("value2"),
				},
			},
			NextToken: nil,
		}
		return result, nil
	}
	return nil, errors.New("error in GetParametersByPath")
}

func Test_GetParameters_Success(t *testing.T) {
	//Arrange
	ssmRepository = InitializeSSMClient(true)

	//Act
	actual, _ := ssmRepository.GetParameters()

	//Assert
	assert.Equal(t, "value1", actual["param1"])
	assert.Equal(t, "value2", actual["param2"])
}

func Test_GetParameters_Failure(t *testing.T) {
	//Arrange
	ssmRepository = InitializeSSMClient(false)
	expected := "error in GetParametersByPath"

	//Act
	_, actual := ssmRepository.GetParameters()

	//Assert
	assert.Equal(t, expected, actual.Error())
}

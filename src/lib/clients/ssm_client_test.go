package clients

import "testing"

func TestNewSSMClient(t *testing.T) {

	//Act
	sfnClient := NewSSMClient(true)

	if sfnClient == nil {
		t.Fail()
	}
}

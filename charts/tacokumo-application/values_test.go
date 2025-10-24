package tacokumoapplication_test

import (
	"os"
	tacokumoapplication "tacokumo/portal-controller-kubernetes/charts/tacokumo-application"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v2"
)

func TestValues(t *testing.T) {
	f, err := os.Open("values.yaml")
	assert.NoError(t, err)
	err = yaml.NewDecoder(f).Decode(&tacokumoapplication.Values{})
	assert.NoError(t, err)

	err = f.Close()
	assert.NoError(t, err)
}

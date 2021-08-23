package pipeline_checker

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/erda-project/kubeprober/probers/erda/control-plane/config"
)

func initEnv() {
	os.Setenv("LOGIN_USER", "dice")
	os.Setenv("LOGIN_PASSWORD", "xxx")
	os.Setenv("SERVICE_NAMESPACE", "project-387-dev")
	os.Setenv("CLUSTER_NAME", "terminus-dev")
	config.Load()
}

func TestLogin(t *testing.T) {
	initEnv()
	c, err := NewChecker()
	assert.NoError(t, err)
	err = c.DoCheck()
	assert.NoError(t, err)
}

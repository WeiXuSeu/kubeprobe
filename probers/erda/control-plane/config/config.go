package config

import (
	"time"

	"github.com/erda-project/kubeprober/pkg/envconf"
)

type Conf struct {
	// deployment service check config
	LoginUser        string `env:"LOGIN_USER" required:"true"`
	LoginPassword    string `env:"LOGIN_PASSWORD" required:"true"`
	ServiceNamespace string `env:"SERVICE_NAMESPACE" default:"default"`
	ClusterName      string `env:"CLUSTER_NAME" required:"true"`
	// sleep time before log check
	LogDelayTime time.Duration `env:"LOG_DELAY_TIME" default:"1m"`

	// common config
	CheckTimeout   time.Duration `env:"CHECK_TIMEOUT" default:"15m"`
	KubeConfigFile string        `env:"KUBECONFIG_FILE"`
	Debug          bool          `env:"DEBUG" default:"false"`
}

var Cfg Conf

// Load 从环境变量加载配置选项.
func Load() {
	envconf.MustLoad(&Cfg)
}

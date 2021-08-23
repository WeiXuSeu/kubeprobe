package main

import (
	"fmt"

	"github.com/sirupsen/logrus"

	proberchecker "github.com/erda-project/kubeprober/pkg/probe-checker"
	"github.com/erda-project/kubeprober/probers/erda/control-plane/config"
	login "github.com/erda-project/kubeprober/probers/erda/control-plane/login_checker"
	pipeline "github.com/erda-project/kubeprober/probers/erda/control-plane/pipeline_checker"
)

func main() {
	var (
		err error
		s   *login.LoginChecker
		d   *pipeline.PipelineChecker
	)

	defer func() {
		if err != nil {
			panic(err)
		}
	}()

	// load config
	config.Load()
	if err != nil {
		err = fmt.Errorf("parse config failed, error: %v", err)
		return
	}
	// check log debug level
	if config.Cfg.Debug {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debug("DEBUG MODE")
	}

	// create checkers
	s, err = login.NewChecker()
	if err != nil {
		err = fmt.Errorf("new deployment service checker failed, error: %v", err)
		return
	}

	d, err = pipeline.NewChecker()
	if err != nil {
		err = fmt.Errorf("new dns checker failed, error: %v", err)
		return
	}

	// run checkers
	err = proberchecker.RunCheckers(proberchecker.CheckerList{s, d})
	if err != nil {
		err = fmt.Errorf("run deployment service checker failed, error: %v", err)
		return
	}
	logrus.Infof("run erda pipeline checker successfully")
}

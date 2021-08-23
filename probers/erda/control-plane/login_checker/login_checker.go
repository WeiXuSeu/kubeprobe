package login_checker

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"

	kubeproberv1 "github.com/erda-project/kubeprober/apis/v1"
	"github.com/erda-project/kubeprober/probers/erda/control-plane/config"
)

type LoginChecker struct {
	Name     string
	Status   kubeproberv1.CheckerStatus
	Timeout  time.Duration
	loginUrl string
	client   *resty.Client
}

func NewChecker() (*LoginChecker, error) {
	c := LoginChecker{
		Name:     "erda-login-check",
		Timeout:  config.Cfg.CheckTimeout,
		loginUrl: fmt.Sprintf("http://openapi.%s.svc.cluster.local:9529/login", config.Cfg.ServiceNamespace),
		client:   resty.New(),
	}
	return &c, nil
}

func (c *LoginChecker) GetName() string {
	return c.Name
}

func (c *LoginChecker) SetName(n string) {
	c.Name = n
}

func (c *LoginChecker) GetStatus() kubeproberv1.CheckerStatus {
	return c.Status
}

func (c *LoginChecker) SetStatus(s kubeproberv1.CheckerStatus) {
	c.Status = s
}

func (c *LoginChecker) GetTimeout() time.Duration {
	return c.Timeout
}

func (c *LoginChecker) SetTimeout(t time.Duration) {
	c.Timeout = t
}

func (c *LoginChecker) DoCheck() error {
	type LoginResponse struct {
		SessionID string `json:"sessionid"`
	}

	lr := LoginResponse{}

	c.client.SetRetryCount(3).SetRetryWaitTime(3 * time.Second).SetRetryMaxWaitTime(20 * time.Second)
	resp, err := c.client.R().
		SetBody(map[string]interface{}{"username": config.Cfg.LoginUser, "password": config.Cfg.LoginPassword}).
		Post(c.loginUrl)

	if err != nil {
		logrus.Errorf("login failed, error: %v", err)
		return err
	}

	if resp.StatusCode() != 200 {
		err := fmt.Errorf("login failed, status code should be 200, but get: %v", resp.StatusCode())
		logrus.Errorf(err.Error())
		return err
	}

	err = json.Unmarshal(resp.Body(), &lr)
	if err != nil {
		logrus.Errorf("unmarshal response body failed, error: %v", err)
		return err
	}

	if lr.SessionID == "" {
		err := fmt.Errorf("login failed, get empty sessionid")
		logrus.Errorf(err.Error())
		return err
	}
	logrus.Infof("login check pass!!!")

	return nil
}

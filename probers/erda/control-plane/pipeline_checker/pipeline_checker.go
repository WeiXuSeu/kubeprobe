package pipeline_checker

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"

	kubeproberv1 "github.com/erda-project/kubeprober/apis/v1"
	"github.com/erda-project/kubeprober/apistructs"
	"github.com/erda-project/kubeprober/probers/erda/control-plane/config"
)

const (
	ErdaTestPipeline = `
version: "1.1"
stages:
- stage:
  - echo:
      alias: pipeline-test
      version: "1.0"
      timeout: 900
      params:
        what: "hello world, pipeline test!"
`
)

type PipelineChecker struct {
	Name               string
	Status             kubeproberv1.CheckerStatus
	Timeout            time.Duration
	pipelineHost       string
	dopHost            string
	pipelineCreatePath string
	pipelineGetPath    string
	client             *resty.Client
}

func NewChecker() (*PipelineChecker, error) {
	c := PipelineChecker{
		Name:               "erda-pipeline-check",
		Timeout:            config.Cfg.CheckTimeout,
		pipelineHost:       fmt.Sprintf("http://pipeline.%s.svc.cluster.local:3081", config.Cfg.ServiceNamespace),
		dopHost:            fmt.Sprintf("http://dop.%s.svc.cluster.local:9527", config.Cfg.ServiceNamespace),
		pipelineCreatePath: "/api/v2/pipelines",
		// post with pipelineID
		pipelineGetPath: "/api/pipelines",
		client:          resty.New(),
	}

	return &c, nil
}

func (c *PipelineChecker) GetName() string {
	return c.Name
}

func (c *PipelineChecker) SetName(n string) {
	c.Name = n
}

func (c *PipelineChecker) GetStatus() kubeproberv1.CheckerStatus {
	return c.Status
}

func (c *PipelineChecker) SetStatus(s kubeproberv1.CheckerStatus) {
	c.Status = s
}

func (c *PipelineChecker) GetTimeout() time.Duration {
	return c.Timeout
}

func (c *PipelineChecker) SetTimeout(t time.Duration) {
	c.Timeout = t
}

func (c *PipelineChecker) DoCheck() (err error) {
	logrus.Infof("start to run checker: %s", c.GetName())
	pID, err := c.CreatePipeline()
	if err != nil {
		err = fmt.Errorf("create pipeline failed, error: %v", err)
		logrus.Errorf(err.Error())
		return
	}
	err = c.WaitPipeline(pID)
	if err != nil {
		err = fmt.Errorf("wait pipeline failed, pipelineID: %v, error: %v", pID, err)
		logrus.Errorf(err.Error())
		return
	}

	time.Sleep(config.Cfg.LogDelayTime)
	err = c.CheckPipelineLog(pID)
	if err != nil {
		err = fmt.Errorf("get pipeline log failed, pipelineID: %v, error: %v", pID, err)
		logrus.Errorf(err.Error())
		return
	}
	return
}

func (c *PipelineChecker) CreatePipeline() (pipelineID int, err error) {
	type Err struct {
		Code    string `json:"code"`
		Message string `json:"msg"`
	}
	type Data struct {
		PipelineID int `json:"id"`
	}
	type CreateResponse struct {
		Success bool `json:"success"`
		Error   Err  `json:"err"`
		Data    Data `json:"data"`
	}

	cr := CreateResponse{}

	c.client.SetRetryCount(3).SetRetryWaitTime(3 * time.Second).SetRetryMaxWaitTime(20 * time.Second)
	resp, err := c.client.R().
		SetHeaders(map[string]string{"Internal-Client": "bundle"}).
		SetBody(map[string]interface{}{
			"pipelineYml":     ErdaTestPipeline,
			"pipelineYmlName": fmt.Sprintf("kubeprober-pipeline-test-%v", time.Now().Unix()),
			"clusterName":     config.Cfg.ClusterName,
			"pipelineSource":  "ops",
			"autoRunAtOnce":   true,
		}).
		Post(strings.Join([]string{c.pipelineHost, c.pipelineCreatePath}, ""))

	if err != nil {
		logrus.Errorf("create pipeline failed, error: %v", err)
		return 0, err
	}

	if resp.StatusCode() != 200 || resp.Error() != nil {
		err := fmt.Errorf("status code: %v, error: %v", resp.StatusCode(), resp.Error())
		logrus.Errorf(err.Error())
		return 0, err
	}

	err = json.Unmarshal(resp.Body(), &cr)
	if err != nil {
		logrus.Errorf("unmarshal response body failed, error: %v", err)
		return 0, err
	}

	if cr.Data.PipelineID <= 0 {
		err := fmt.Errorf("create pipeline failed, invalid pipeline id: %v, error: %v", cr.Data.PipelineID, cr.Error)
		logrus.Errorf(err.Error())
		return 0, err
	}
	logrus.Infof("created pipeline: %v", cr.Data.PipelineID)

	return cr.Data.PipelineID, nil
}

func (c *PipelineChecker) WaitPipeline(pipelineID int) error {
	for i := 0; i < 15; i++ {
		time.Sleep(1 * time.Minute)
		unfinished, _, err := c.GetPipeline(pipelineID)
		if err != nil {
			logrus.Errorf("get pipeline failed, pipelineID: %v, error: %v", pipelineID, err)
			return err
		}
		if !unfinished {
			return nil
		}
	}
	return nil
}

func (c *PipelineChecker) GetPipeline(pipelineID int) (unfinished bool, dto *apistructs.PipelineDetailDTO, err error) {

	resp, err := c.client.SetRetryCount(3).
		SetRetryWaitTime(3 * time.Second).
		SetRetryMaxWaitTime(20 * time.Second).
		R().
		SetHeaders(map[string]string{"Internal-Client": "bundle"}).
		Get(fmt.Sprintf("%s%s/%v", c.pipelineHost, c.pipelineGetPath, pipelineID))

	if err != nil {
		logrus.Errorf("pipelineID: %v, error: %v", pipelineID, err)
		return
	}

	if resp.StatusCode() != 200 || resp.Error() != nil {
		err = fmt.Errorf("status code: %v, error: %v", resp.StatusCode(), resp.Error())
		logrus.Errorf(err.Error())
		return
	}
	gpr := apistructs.GetPipelineResponse{}

	err = json.Unmarshal(resp.Body(), &gpr)
	if err != nil {
		logrus.Errorf("unmarshal response body failed, error: %v", err)
		return
	}

	if gpr.Error.Msg != "" || !gpr.Success {
		err = fmt.Errorf("error: %v", gpr.Error)
		return
	}

	dto = &gpr.Data

	if len(dto.PipelineStages) == 0 {
		err = fmt.Errorf("len(dto.PipelineStages) == 0, pipelineid: %d", pipelineID)
		logrus.Errorf(err.Error())
		return
	}

	if len(dto.PipelineStages[0].PipelineTasks) == 0 {
		err = fmt.Errorf("len(dto.PipelineStages[0].PipelineTasks) == 0, pipelineid: %d", pipelineID)
		logrus.Errorf(err.Error())
		return
	}

	for _, stage := range dto.PipelineStages {
		for _, task := range stage.PipelineTasks {
			errStr := getPipelineErrMsg(task.Result.Errors)
			if errStr != "" {
				err = fmt.Errorf("%s", errStr)
				return
			}
			if task.Status.IsFailedStatus() {
				err = fmt.Errorf("status: %v", task.Status)
				return
			}
			if !task.Status.IsSuccessStatus() {
				logrus.Infof("pipeline still running, status: %v", task.Status)
				unfinished = true
				return
			}
		}
	}

	logrus.Infof("pipeline finished: %+v", dto)
	return
}

func getPipelineErrMsg(errs []apistructs.ErrorResponse) string {
	var result string
	for _, err := range errs {
		result = fmt.Sprintf("%s; %s", result, err.Msg)
	}
	return result
}

func (c *PipelineChecker) CheckPipelineLog(pipelineID int) (err error) {
	type Log struct {
		Content string `json:"content"`
	}

	type LogLines struct {
		Lines []Log `json:"lines"`
	}

	type LogResponse struct {
		apistructs.Header
		Data LogLines `json:"data"`
	}

	_, dto, err := c.GetPipeline(pipelineID)
	if err != nil {
		logrus.Errorf("get pipeline failed, pipelineID: %v, error: %v", pipelineID, err)
		return
	}

	if len(dto.PipelineStages) == 0 {
		err = fmt.Errorf("len(dto.PipelineStages) == 0, pipelineid: %d", pipelineID)
		logrus.Errorf(err.Error())
		return
	}

	if len(dto.PipelineStages[0].PipelineTasks) == 0 {
		err = fmt.Errorf("len(dto.PipelineStages[0].PipelineTasks) == 0, pipelineid: %d", pipelineID)
		logrus.Errorf(err.Error())
		return
	}

	taskID := dto.PipelineStages[0].PipelineTasks[0].ID
	taskLogPath := fmt.Sprintf("/api/cicd/%d/tasks/%d/logs", pipelineID, taskID)

	resp, err := c.client.SetRetryCount(3).
		SetRetryWaitTime(10*time.Second).
		SetRetryMaxWaitTime(60*time.Second).
		R().
		SetQueryParam("source", "job").
		SetQueryParam("start", "0").
		SetQueryParam("count", "-2").
		SetHeaders(map[string]string{"Internal-Client": "bundle"}).
		Get(strings.Join([]string{c.dopHost, taskLogPath}, ""))

	if err != nil {
		err := fmt.Errorf("query log failed, pipelineID: %v, taskID: %v, error: %v", pipelineID, taskID, err)
		logrus.Errorf(err.Error())
		return err
	}

	if resp.StatusCode() != 200 || resp.Error() != nil {
		err = fmt.Errorf("status code: %v, error: %v", resp.StatusCode(), resp.Error())
		logrus.Errorf(err.Error())
		return
	}

	lr := LogResponse{}
	err = json.Unmarshal(resp.Body(), &lr)
	if err != nil {
		logrus.Errorf("unmarshal response body failed, error: %v", err)
		return
	}

	if lr.Error.Msg != "" || !lr.Success {
		err = fmt.Errorf("error: %v", lr.Error)
		return
	}

	isLogEmpty := true
	logContent := ""
	for _, c := range lr.Data.Lines {
		if c.Content != "" {
			isLogEmpty = false
			logContent = c.Content
		}
	}

	if isLogEmpty {
		err = fmt.Errorf("pipeline task log is empty")
		return
	}

	logrus.Infof("get pipeline log successfully, log content: %s", logContent)

	return nil
}

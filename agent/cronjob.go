package agent

import (
	"time"

	"github.com/gorhill/cronexpr"
	log "github.com/ngaut/logging"
)

var (
	// what to do when Job timeout
	TriggerIgnore int = 0
	TriggerKill   int = 1

	JobSuccess int = 0
	JobFail    int = 1
	JobTimeout int = 2
	JobRunning int = 3
	JobKilled  int = 4
)

// job for crontab
type CronJob struct {
	Id               int64  `json:"id"`              // job id generated by mysql autoincrement
	Name             string `json:"name"`            // job name
	CreateUser       string `json:"create_user"`     // create by user
	Executor         string `json:"executor"`        // job execute file
	ExecutorFlags    string `json:"executor_flags"`  // job execute file args
	Signature        string `json:"signature"`       // signature for executor by md5sum
	Runner           string `json:"runner"`          // su - runner to run, root is forbidden
	Timeout          int64  `json:"timeout"`         // timeout
	OnTimeoutTrigger int    `json:"timeout_trigger"` // on timeout trigger
	Disabled         bool   `json:"disabled"`        // we don't schedule this job when disabled
	Schedule         string `json:"schedule"`        // schedule crontab format
	WebHookUrl       string `json:"hook"`            // we post status to this webhookurl
	MsgFilter        string `json:"msg_filter"`      // when stdout contain msg filter
	// CreateAt will modify when cronjob update
	// so when we compare Agent's Jobs and Store' Jobs, CreateAt newer will win and Alert
	CreateAt int64 `json:"create_at"` // create timestamp

	expression *cronexpr.Expression `json:"-"` // expression generated by Schedule
	Dcms       *Agent               `json:"-"` // Dcms this CronJob belong to

	SuccessCnt    int    `json:"success_count"`   // success count
	ErrCnt        int    `json:"error_count"`     // failed count
	TimeoutCnt    int    `json:"timeout_count"`   // timeout count (update success or failed count)
	LastTaskId    string `json:"last_task_id"`    // last running time ID auto increment
	LastSuccessAt int64  `json:"last_success_at"` // last success ts
	LastErrAt     int64  `json:"last_error_at"`   //last err ts
	LastStatus    int    `json:"last_status"`     //latest new status
	LastExecAt    int64  `json:"last_exec_at"`
}

// check crontab need
func (cj *CronJob) NeedSchedule() bool {
	if cj.Disabled {
		return false
	}

	expression, err := cronexpr.Parse(cj.Schedule)
	if err != nil {
		log.Warning("crontab parse failed: ", cj.Id, cj.Name)
		cj.Disabled = true
	}
	cj.expression = expression

	if cj.LastExecAt == 0 {
		cj.LastExecAt = cj.CreateAt
	}

	last_run_time := time.Unix(cj.LastExecAt, 0)
	nt := cj.expression.Next(last_run_time)
	// log.Info("cron next run is: ", cj.Id, cj.Name, nt)
	if time.Now().Unix()-nt.Unix() > 20 {
		// log.Info("needschedule true: ", cj.Id, cj.Name, nt)
		return true
	}
	// log.Info("needschedule false: ", cj.Id, cj.Name, nt)
	return false
}

// exec this function when task for this job timeout
func (cj *CronJob) OnTimeout() int {
	return cj.OnTimeoutTrigger
}

// must check crontab  valid  first
func (cj *CronJob) IsValid() bool {
	if _, err := cronexpr.Parse(cj.Schedule); err != nil {
		log.Warning("cron job parse crontab format failed: ", err)
		return false
	}
	if cj.Runner == "root" {
		log.Warning("cron job must run under non-root user, current is: ", cj.Runner)
		return false
	}
	return !cj.Disabled
}

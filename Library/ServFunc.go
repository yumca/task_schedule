package Library

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

type WorkerStat struct {
	TaskKey    string
	Runtime    string
	Status     string
	WorkerType string
}

type ExecOutput struct {
	Output       string
	StartRuntime string
	EndRuntime   string
}

type ServFunc struct {
	SysRun       string
	AsyncRun     int
	RunWorker    map[string]string
	RunWorkerNum int
	WorkerStat   map[string]WorkerStat
	WorkerChan   map[string]chan Command
	// WorkerChanData map[string]map[string]Command
	Config       Config
	RdbWorkerKey string
	rdb          *redis.Client
	MyLog        *MyLogs
}

func NewServFunc(conf Config) (Serv *ServFunc, err error) {
	Serv = nil
	Serv = &ServFunc{Config: conf}
	if conf.Redis.Stat == "on" {
		rdb, err := Rdlink()
		if err == nil {
			Serv.RdbWorkerKey = conf.Redis.RedisPrefix + conf.Setting.WorkerName + ":Worker"
			Serv.rdb = rdb
		}
	}
	return
}

// Start 开始程序
func (s *ServFunc) Start() error {
	fmt.Println("ServFunc Start")
	//初始化系统运行状态
	s.SysRun = "running"
	//初始化已运行进程数量
	s.RunWorker = make(map[string]string)
	//初始化已运行进程数量
	s.RunWorkerNum = 0
	//初始化异步任务执行协程数量
	s.AsyncRun = 0
	//初始化运行的worker状态
	s.WorkerStat = make(map[string]WorkerStat)
	//初始化运行的worker通道
	s.WorkerChan = make(map[string]chan Command)
	//初始化运行的worker执行列表状态
	// s.WorkerChanData = make(map[string]map[string]Command)
	//开启日志插件
	mylog, err := NewMyLogs("logs", "task_schedule", "", nil)
	if err != nil {
		return err
	}
	s.MyLog = mylog
	//开启定时协程
	go s.tick()
	//清除非配置中的Worker
	if s.Config.Redis.Stat == "on" {
		keys := s.rdb.Keys(ctx, s.RdbWorkerKey+":*")
		for _, v := range keys.Val() {
			tmp := strings.Split(v, ":")
			if _, ok := s.Config.Worker[tmp[2]]; !ok {
				s.rdb.Del(ctx, s.RdbWorkerKey+":"+tmp[2])
			}
		}
		//开启getCmdForRdb获取redis队列数据协程
		go s.getCmdForRdb()
	}
	//添加任务协程状态数据
	for _, v := range s.Config.Worker {
		if s.Config.Redis.Stat == "on" {
			s.rdb.HSet(ctx, s.RdbWorkerKey+":"+v.TaskKey, "TaskKey", v.TaskKey)
			s.rdb.HSet(ctx, s.RdbWorkerKey+":"+v.TaskKey, "Runtime", time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05"))
			s.rdb.HSet(ctx, s.RdbWorkerKey+":"+v.TaskKey, "Status", v.Status)
			s.rdb.HSet(ctx, s.RdbWorkerKey+":"+v.TaskKey, "WorkerType", v.WorkerType)
		}
		if v.Status == "running" {
			//开启任务协程
			if v.WorkerType == "sync" {
				go s.sync(v)
			} else if v.WorkerType == "async" {
				go s.async(v)
			}
		}
	}
	fmt.Println("ServFunc running")
	return nil
}

//同步任务协程
func (s *ServFunc) sync(Worker WorkerInfo) {
	fmt.Printf("同步任务协程：taskKey-%v开始;\n", Worker.TaskKey)
	s.RunWorkerNum++
	s.RunWorker[Worker.TaskKey] = "running"
	s.WorkerStat[Worker.TaskKey] = WorkerStat{
		TaskKey:    Worker.TaskKey,
		Runtime:    time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05"),
		Status:     Worker.Status,
		WorkerType: Worker.WorkerType,
	}
	s.WorkerChan[Worker.TaskKey] = make(chan Command)
	for c := range s.WorkerChan[Worker.TaskKey] {
		output, err := s.execTaskCommand(c)
		if err != nil {
			s.MyLog.DoLogs(fmt.Sprintf("同步队列任务执行失败：%v;", err), "e", "task", nil)
		}
		s.MyLog.DoLogs("同步队列任务执行完毕", "n", "task", map[string]string{
			"startRuntime": output.StartRuntime,
			"endRuntime":   output.EndRuntime,
			"output":       output.Output,
		})
	}
	fmt.Printf("同步任务协程：taskKey-%v退出;\n", Worker.TaskKey)
	s.RunWorkerNum--
	s.RunWorker[Worker.TaskKey] = "stop"
}

//异步任务协程
func (s *ServFunc) async(Worker WorkerInfo) {
	fmt.Printf("异步任务协程：taskKey-%s开始;\n", Worker.TaskKey)
	s.RunWorkerNum++
	s.RunWorker[Worker.TaskKey] = "running"
	s.WorkerStat[Worker.TaskKey] = WorkerStat{
		TaskKey:    Worker.TaskKey,
		Runtime:    time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05"),
		Status:     Worker.Status,
		WorkerType: Worker.WorkerType,
	}
	s.WorkerChan[Worker.TaskKey] = make(chan Command)
	for c := range s.WorkerChan[Worker.TaskKey] {
		if len(c.Script) > 1 && c.Script != "" {
			go s.doasync(c)
		}
	}
	fmt.Printf("异步任务协程：taskKey-%s退出;\n", Worker.TaskKey)
	s.RunWorkerNum--
	s.RunWorker[Worker.TaskKey] = "stop"
}

//执行异步任务协程
func (s *ServFunc) doasync(taskCommand Command) {
	s.AsyncRun++
	output, err := s.execTaskCommand(taskCommand)
	if err != nil {
		s.MyLog.DoLogs(fmt.Sprintf("异步队列任务执行失败：%v;", err), "e", "task", nil)
		//fmt.Printf("异步队列任务执行失败：%v;\n", err)
	}
	s.MyLog.DoLogs("异步队列任务执行完毕", "n", "task", map[string]string{
		"startRuntime": output.StartRuntime,
		"endRuntime":   output.EndRuntime,
		"output":       output.Output,
	})
	s.AsyncRun--
	//fmt.Printf("异步队列任务执行开始时间：%v;\n", startRuntime)
	//fmt.Printf("异步队列任务执行结束时间：%v;\n", endRuntime)
	//fmt.Printf("异步队列任务执行返回：%v;\n", output)
}

//执行任务
func (s ServFunc) execTaskCommand(taskCommand Command) (output ExecOutput, err error) {
	err = nil
	//判断脚本是否为空
	if taskCommand.Script == "" {
		err = errors.New("Script为空")
		return
	}
	if s.Config.BinPath[taskCommand.TaskType] == "" {
		err = errors.New("BinPath为空")
		return
	}
	if taskCommand.Expire == "" {
		taskCommand.Expire = "1m"
	}
	execStr := ""
	commandName := ""
	if runtime.GOOS == "windows" {
		commandName = s.Config.BinPath[taskCommand.TaskType]
		execStr = taskCommand.Script
	} else {
		commandName = "timeout"
		execStr = taskCommand.Expire + " " + taskCommand.Script + " " + s.Config.BinPath[taskCommand.TaskType]
	}
	if len(taskCommand.Param) > 0 && taskCommand.Param != "" {
		typeof := reflect.TypeOf(taskCommand.Param).Kind().String()
		var param string
		if typeof == "array" || typeof == "map" || typeof == "interface" || typeof == "struct" {
			jsonBytes, errParam := json.Marshal(taskCommand.Param)
			if errParam != nil {
				err = errParam
				return
			}
			param = string(jsonBytes)
		} else {
			param = fmt.Sprintf("%v", taskCommand.Param)
		}
		//对参数进行url转义
		execStr += " " + url.QueryEscape(param)
	}
	startRuntime := time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05")
	execArr := TrimEmpty(strings.Split(execStr, " "))
	var execCmd *exec.Cmd
	if runtime.GOOS == "windows" {
		execCmd = exec.Command(commandName, execArr...)
	} else {
		execCmd = exec.Command(commandName, execArr...)
	}
	execoutput, erroutput := execCmd.Output()
	endRuntime := time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05")
	if erroutput != nil {
		err = erroutput
		// s.MyLog.DoLogs(fmt.Sprintf("异步队列任务执行失败：%v;", err), "e", "task", nil)
		//fmt.Printf("异步队列任务执行失败：%v;\n", err)
		return
	}
	output = ExecOutput{
		EndRuntime:   endRuntime,
		StartRuntime: startRuntime,
		Output:       string(execoutput),
	}
	//成功则删除命令数据
	if s.Config.Redis.Stat == "on" {
		s.rdb.HDel(ctx, taskCommand.TaskKey+"_command", taskCommand.UniqueTaskId)
	}
	return
}

//获取redis队列数据
func (s *ServFunc) getCmdForRdb() {
	println("getCmdForRdb start")
	var taskCommand Command
	for {
		//获取队列信息
		task := s.rdb.BLPop(ctx, time.Second, s.RdbWorkerKey+":CommonList").Val()
		if len(task) > 1 && task[1] != "" {
			taskCommand.CronTask = false
			err := json.Unmarshal([]byte(task[1]), &taskCommand)
			if err != nil {
				s.MyLog.DoLogs(fmt.Sprintf("获取redis队列数据协程-json解析command错误：command-%v,err-%v;", task[1], err), "e", "task", nil)
				continue
			}
			if taskCommand.TaskKey == "" {
				s.MyLog.DoLogs(fmt.Sprintf("获取redis队列数据协程-common缺少必要参数：command-%v;", task[1]), "e", "task", nil)
				continue
			}
			s.WorkerChan[taskCommand.TaskKey] <- taskCommand
		}
		if s.SysRun == "running" {
			continue
		}
		break
	}
	fmt.Printf("获取redis队列数据协程退出;\n")
}

//定时任务方法
func (s *ServFunc) tick() {
	println("tick start")
	ticker := time.NewTicker(time.Second)
	//ticker := time.NewTicker(time.Millisecond)
	for range ticker.C { //遍历ticker.C，如果有值，则会执行，否则阻塞
		if s.SysRun != "running" {
			ticker.Stop()
		}
		limitTime := []int{
			time.Now().Second(),
			time.Now().Minute(),
			time.Now().Hour(),
			time.Now().Day(),
			int(time.Now().Month()),
			time.Now().Year(),
		}
		//每次检查cron数据是否更改
		stat, cron, cronRtime, _ := CheckCron(s.Config.ConfPath, s.Config.CronRtime)
		if stat {
			s.Config.Cron = cron
			s.Config.CronRtime = cronRtime
		}
		for _, cronv := range s.Config.Cron {
			if cronv.TaskKey != "" && s.RunWorker[cronv.TaskKey] == "running" {
				if r := s.tickCheck(cronv.RunTime, limitTime); r {
					command := Command{
						CronTask:     true,
						TaskKey:      cronv.TaskKey,
						TaskType:     cronv.TaskType,
						Expire:       cronv.Expire,
						RunTime:      time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05"),
						Script:       cronv.Script,
						Param:        cronv.Param,
						UniqueTaskId: cronv.UniqueTaskId + s.uniqueTaskId(),
					}
					// s.WorkerChanData[cronv.TaskKey][command.UniqueTaskId] = command
					if s.Config.Redis.Stat == "on" {
						jsonBytes, err := json.Marshal(command)
						if err != nil {
							s.MyLog.DoLogs("缓存定时任务错误,json化失败;", "e", "task", command)
						} else {
							tickcmd := s.rdb.HSet(ctx, command.TaskKey+"_command", command.UniqueTaskId, string(jsonBytes))
							if tickcmd.Val() != 1 || tickcmd.Err() != nil {
								s.MyLog.DoLogs("缓存定时任务失败;", "e", "task", command)
							}
						}
					}
					s.WorkerChan[cronv.TaskKey] <- command
					//fmt.Printf("投递定时任务:taskKey-%v,uniqueTaskId-%v;\n", command["taskKey"], command["uniqueTaskId"])
				}
			}
		}
	}
	println("tick end")
}

//定时任务时间检查
func (s ServFunc) tickCheck(runTime string, limitTime []int) (r bool) {
	r = true
	runTimeArr := TrimEmpty(strings.Split(runTime, " "))
	if len(runTimeArr) < 5 {
		return
	}
	for i := 0; i < 5; i++ {
		runRes := false
		if runTimeArr[i] == "*" {
			runRes = true
		} else {
			var runTime0 []string
			if find := strings.Contains(runTimeArr[i], ","); find {
				runTime0 = TrimEmpty(strings.Split(runTimeArr[i], ","))
			} else {
				runTime0 = []string{runTimeArr[i]}
			}
			if runTime0[0] == "" {
				return
			}
			for _, v := range runTime0 {
				if find := strings.Contains(v, "/"); find {
					if tmp := TrimEmpty(strings.Split(v, "/")); tmp[0] == "*" {
						if s, _ := strconv.Atoi(tmp[1]); (limitTime[i] % s) == 0 {
							runRes = true
							break
						}
					}
				} else if find := strings.Contains(v, "-"); find {
					tmp := TrimEmpty(strings.Split(v, "-"))
					s1, _ := strconv.Atoi(tmp[0])
					s2, _ := strconv.Atoi(tmp[0])
					if s1 <= limitTime[i] && s2 >= limitTime[i] {
						runRes = true
						break
					}
				} else {
					s, _ := strconv.Atoi(v)
					if s == limitTime[i] {
						runRes = true
						break
					}
				}
			}
		}
		if !runRes {
			r = false
			break
		}
	}
	return
}

/**
 * 关闭通道
 */
func (s *ServFunc) stopWorkerChan() {
	fmt.Println("等待关闭队列通道")
	for k, v := range s.WorkerChan {
		i := 0
		for {
			if len(v) < 1 {
				break
			}
			if i > 60 {
				break
			}
			time.Sleep(time.Millisecond * 500)
			i++
		}
		close(v)
		fmt.Printf("队列通道%s已关闭;\n", k)
	}
	fmt.Println("队列通道已全部关闭;")
}

func (s *ServFunc) ExitServ() {
	fmt.Println("开始结束程序...")
	//更新系统运行状态
	s.SysRun = "stop"
	//关闭携程通道
	s.stopWorkerChan()
	//更新redis任务协程状态数据
	if s.Config.Redis.Stat == "on" {
		for _, v := range s.Config.Worker {
			s.rdb.HSet(ctx, s.RdbWorkerKey+":"+v.TaskKey, "Status", "stop")
		}
	}
	//关闭日志通道
	s.MyLog.stop()
	sec := 0
	//确定同/异步任务协程是否执行完毕并关闭
	for s.RunWorkerNum > 0 {
		fmt.Printf("等待同/异步任务协程关闭,还有%d个未关闭;\n", s.RunWorkerNum)
		time.Sleep(time.Second)
		sec++
		if sec >= 60 {
			break
		}
	}
	//确定异步协程任务是否执行完毕
	for s.AsyncRun > 0 {
		fmt.Printf("等待异步协程任务执行完毕,还有%d个未执行完毕;\n", s.AsyncRun)
		time.Sleep(time.Second)
		sec++
		if sec >= 60 {
			fmt.Printf("等待异步协程任务执行超时60s,还有%d个未执行完毕;直接结束\n", s.AsyncRun)
			break
		}
	}
	fmt.Println("结束程序...")
	os.Exit(0)
}

//获取唯一md5的任务id
func (s ServFunc) uniqueTaskId() string {
	data := []byte(strconv.Itoa(int(time.Now().UnixNano())))
	h := md5.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(data))
}

package Library

import (
    "crypto/md5"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "github.com/go-redis/redis/v8"
    "net/url"
    "os"
    "os/exec"
    "reflect"
    "runtime"
    "strconv"
    "strings"
    "time"
)

type command struct {
    cronTask     int
    taskKey      string
    script       string
    controller   string
    function     string
    param        interface{}
    uniqueTaskId string
}

type ServFunc struct {
    SysRun     string
    AsyncRun   int
    RunWorker  int
    WorkerStat map[string]string
    WorkerKey  string
    Config     Config
    rdb        *redis.Client
    MyLog      *MyLogs
}

func NewServFunc(conf Config) (Serv *ServFunc, err error) {
    Serv = nil
    rdb, err := Rdlink()
    if err != nil {
        return
    }
    workerKey := conf.Redis.RedisPrefix + conf.Setting.WorkerName + "Task:Worker"
    Serv = &ServFunc{Config: conf, rdb: rdb, WorkerKey: workerKey}
    return
}

// Start 开始程序
func (s *ServFunc) Start() error {
    fmt.Println("ServFunc Start")
    //初始化系统运行状态
    s.SysRun = "running"
    //初始化已运行进程数量
    s.RunWorker = 0
    //初始化异步任务执行协程数量
    s.AsyncRun = 0
    //初始化运行的worker状态
    s.WorkerStat = make(map[string]string)
    //清除非配置中的Worker
    keys := s.rdb.Keys(ctx, s.WorkerKey+":*")
    for _, v := range keys.Val() {
        tmp := strings.Split(v, ":")
        if _, ok := s.Config.Worker[tmp[2]]; !ok {
            s.rdb.Del(ctx, s.WorkerKey+":"+tmp[2])
        }
    }
    //开启定时协程
    go s.tick()
    //添加任务协程状态数据
    for _, v := range s.Config.Worker {
        s.rdb.HSet(ctx, s.WorkerKey+":"+v.TaskKey, "task_key", v.TaskKey)
        s.rdb.HSet(ctx, s.WorkerKey+":"+v.TaskKey, "run_time", time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05"))
        s.rdb.HSet(ctx, s.WorkerKey+":"+v.TaskKey, "status", v.Status)
        if v.Status == "running" {
            //开启任务协程
            if v.WorkerType == "sync" {
                go s.sync(v.TaskKey)
            } else if v.WorkerType == "async" {
                go s.async(v.TaskKey)
            }
        }
    }
    //开启日志插件
    mylog, err := NewMyLogs("logs", "task_schedule", "", nil)
    if err != nil {
        return err
    }
    s.MyLog = mylog
    fmt.Println("ServFunc running")
    return nil
}

//同步任务协程
func (s *ServFunc) sync(taskKey string) {
    fmt.Printf("同步任务协程：taskKey-%v开始;\n", taskKey)
    s.RunWorker++
    s.WorkerStat[taskKey] = "running"
    var execCmd *exec.Cmd
    for {
        //获取队列信息
        task := s.rdb.BLPop(ctx, time.Second, taskKey).Val()
        if len(task) > 1 && task[1] != "" {
            //获取命令数据
            taskCommand := make(map[string]string)
            tmpCommand := s.rdb.HGet(ctx, taskKey+"_command", task[1]).Val()
            err := json.Unmarshal([]byte(tmpCommand), &taskCommand)
            if err != nil {
                continue
            }
            //删除命令数据
            s.rdb.HDel(ctx, taskKey+"_command", task[1])
            if taskCommand["script"] == "" {
                continue
            }
            if s.Config.BinPath[taskCommand["TaskType"]] == "" {
                continue
            }
            if taskCommand["Expire"] == "" {
                taskCommand["Expire"] = "1m"
            }
            execStr := ""
            commandName := ""
            if runtime.GOOS == "windows" {
                commandName = s.Config.BinPath[taskCommand["TaskType"]]
                execStr = taskCommand["script"]
            } else {
                commandName = "timeout"
                execStr = taskCommand["Expire"] + " " + taskCommand["script"] + " " + s.Config.BinPath[taskCommand["TaskType"]]
            }
            if len(taskCommand["param"]) > 0 && taskCommand["param"] != "" {
                typeof := reflect.TypeOf(taskCommand["param"]).Kind().String()
                var param string
                if typeof == "array" || typeof == "map" || typeof == "interface" || typeof == "struct" {
                    jsonBytes, err := json.Marshal(taskCommand["param"])
                    if err != nil {
                        continue
                    }
                    param = string(jsonBytes)
                } else {
                    param = fmt.Sprintf("%v", taskCommand["param"])
                }
                //对参数进行url转义
                execStr += " " + url.QueryEscape(param)
            }
            startRuntime := time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05")
            execArr := TrimEmpty(strings.Split(execStr, " "))
            if runtime.GOOS == "windows" {
                execCmd = exec.Command(commandName, execArr...)
            } else {
                execCmd = exec.Command(commandName, execArr...)
            }
            output, err := execCmd.Output()
            endRuntime := time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05")
            if err != nil {
                s.MyLog.DoLogs(fmt.Sprintf("同步队列任务执行失败：%v;", err), "e", "task", nil)
            }
            //logData := map[string]string{
            //    "startRuntime": startRuntime,
            //    "endRuntime":   endRuntime,
            //    "output":       string(output),
            //}
            s.MyLog.DoLogs("同步队列任务执行完毕", "n", "task", map[string]string{
                "startRuntime": startRuntime,
                "endRuntime":   endRuntime,
                "output":       string(output),
            })
            //fmt.Printf("同步队列任务执行开始时间：%v;\n", startRuntime)
            //fmt.Printf("同步队列任务执行结束时间：%v;\n", endRuntime)
            //fmt.Printf("同步队列任务执行返回：%v;\n", string(output))
        }
        if s.WorkerStat[taskKey] == "running" {
            continue
        }
        break
    }
    //s.rdb.HSet(ctx, s.WorkerKey+":"+taskKey, "status", "stop")
    fmt.Printf("同步任务协程：taskKey-%v退出;\n", taskKey)
    s.RunWorker--
}

//异步任务协程
func (s *ServFunc) async(taskKey string) {
    fmt.Printf("异步任务协程：taskKey-%v开始;\n", taskKey)
    s.RunWorker++
    s.WorkerStat[taskKey] = "running"
    for {
        //获取队列信息
        task := s.rdb.BLPop(ctx, time.Second, taskKey).Val()
        if len(task) > 1 && task[1] != "" {
            go s.doasync(taskKey, task[1])
        }
        if s.WorkerStat[taskKey] == "running" {
            continue
        }
        break
    }
    //s.rdb.HSet(ctx, s.WorkerKey+":"+taskKey, "status", "stop")
    fmt.Printf("异步任务协程：taskKey-%v退出;\n", taskKey)
    s.RunWorker--
}

//执行异步任务协程
func (s *ServFunc) doasync(taskKey string, taskId string) {
    s.AsyncRun++
    var execCmd *exec.Cmd
    //获取命令数据
    var taskCommand map[string]string
    err := json.Unmarshal([]byte(s.rdb.HGet(ctx, taskKey+"_command", taskId).Val()), &taskCommand)
    if err != nil {
        s.MyLog.DoLogs(fmt.Sprintf("异步任务协程-json解析command错误：taskKey-%v,uniqueTaskId-%v,err-%v;", taskKey, taskId, err), "e", "task", nil)
        //fmt.Printf("异步任务协程-json解析command错误：taskKey-%v,uniqueTaskId-%v,err-%v;\n", taskKey, taskId, err)
        return
    }
    //删除命令数据
    s.rdb.HDel(ctx, taskKey+"_command", taskId)
    if taskCommand["script"] == "" {
        return
    }
    if s.Config.BinPath[taskCommand["TaskType"]] == "" {
        return
    }
    if taskCommand["Expire"] == "" {
        taskCommand["Expire"] = "1m"
    }
    execStr := ""
    commandName := ""
    if runtime.GOOS == "windows" {
        commandName = s.Config.BinPath[taskCommand["TaskType"]]
        execStr = taskCommand["script"]
    } else {
        commandName = "timeout"
        execStr = taskCommand["Expire"] + " " + taskCommand["script"] + " " + s.Config.BinPath[taskCommand["TaskType"]]
    }
    if len(taskCommand["param"]) > 0 && taskCommand["param"] != "" {
        typeof := reflect.TypeOf(taskCommand["param"]).Kind().String()
        var param string
        if typeof == "array" || typeof == "map" || typeof == "interface" || typeof == "struct" {
            jsonBytes, err := json.Marshal(taskCommand["param"])
            if err != nil {
                return
            }
            param = string(jsonBytes)
        } else {
            param = fmt.Sprintf("%v", taskCommand["param"])
        }
        //对参数进行url转义
        execStr += " " + url.QueryEscape(param)
    }
    startRuntime := time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05")
    execArr := TrimEmpty(strings.Split(execStr, " "))
    if runtime.GOOS == "windows" {
        execCmd = exec.Command(commandName, execArr...)
    } else {
        execCmd = exec.Command(commandName, execArr...)
    }
    output, err := execCmd.Output()
    endRuntime := time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05")
    s.AsyncRun--
    if err != nil {
        s.MyLog.DoLogs(fmt.Sprintf("异步队列任务执行失败：%v;", err), "e", "task", nil)
        //fmt.Printf("异步队列任务执行失败：%v;\n", err)
        return
    }
    s.MyLog.DoLogs("异步队列任务执行完毕", "n", "task", map[string]string{
        "startRuntime": startRuntime,
        "endRuntime":   endRuntime,
        "output":       string(output),
    })
    //fmt.Printf("异步队列任务执行开始时间：%v;\n", startRuntime)
    //fmt.Printf("异步队列任务执行结束时间：%v;\n", endRuntime)
    //fmt.Printf("异步队列任务执行返回：%v;\n", output)
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
        if stat == true {
            s.Config.Cron = cron
            s.Config.CronRtime = cronRtime
        }
        for _, cronv := range s.Config.Cron {
            status := s.rdb.HGet(ctx, s.WorkerKey+":"+cronv.TaskKey, "status").Val()
            //fmt.Printf("队列信息:cron-%v;key-%s;status-%v\n", cronv, s.WorkerKey+":"+cronv.TaskKey, status)
            if cronv.TaskKey != "" && status == "running" {
                if r := s.tickCheck(cronv.RunTime, limitTime); r == true {
                    command := map[string]string{
                        "cronTask":     "1",
                        "taskKey":      cronv.TaskKey,
                        "TaskType":     cronv.TaskType,
                        "Expire":       cronv.Expire,
                        "script":       cronv.Script,
                        "param":        cronv.Param,
                        "uniqueTaskId": cronv.UniqueTaskId + s.uniqueTaskId(),
                    }
                    jsonBytes, err := json.Marshal(command)
                    if err != nil {
                        continue
                    }
                    //fmt.Printf("投递定时任务内容：command-%v;json-%v;\n", command, string(jsonBytes))
                    tickcmd := s.rdb.HSet(ctx, command["taskKey"]+"_command", command["uniqueTaskId"], string(jsonBytes))
                    if tickcmd.Val() != 1 || tickcmd.Err() != nil {
                        s.MyLog.DoLogs("投递定时任务错误;", "e", "task", command)
                        //fmt.Printf("投递定时任务错误;\n")
                        continue
                    }
                    s.rdb.RPush(ctx, command["taskKey"], command["uniqueTaskId"])
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
        if runRes == false {
            r = false
            break
        }
    }
    return
}

func (s *ServFunc) ExitServ() {
    fmt.Println("开始结束程序...")
    //更新系统运行状态
    s.SysRun = "stop"
    //更新任务协程状态数据
    for _, v := range s.Config.Worker {
        s.WorkerStat[v.TaskKey] = "stop"
        s.rdb.HSet(ctx, s.WorkerKey+":"+v.TaskKey, "status", "stop")
    }
    //关闭日志通道
    s.MyLog.stop()
    sec := 0
    //确定同/异步任务协程是否执行完毕并关闭
    for s.RunWorker > 0 {
        fmt.Printf("等待同/异步任务协程关闭,还有%d个未关闭;\n", s.RunWorker)
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

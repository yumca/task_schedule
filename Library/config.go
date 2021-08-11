package Library

import (
    "encoding/json"
    "errors"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
)

type db struct {
    DbHost   string
    DbUser   string
    DbPwd    string
    DbPort   int
    DbPrefix string
}

type rd struct {
    RedisHost   string
    RedisPort   string
    RedisPrefix string
    RedisPwd    string
    RedisDb     int
}

type setting struct {
    WorkerName string
    LogFile    string
    PidFile    string
    Daemonize  int
}

type crontab struct {
    TaskKey      string
    TaskType     string
    Expire       string
    RunTime      string
    Script       string
    Param        string
    UniqueTaskId string
}

type workerInfo struct {
    WorkerType string
    TaskKey    string
    Status     string
}

type Config struct {
    Db        db
    Redis     rd
    Setting   setting
    Cron      []crontab
    Worker    map[string]workerInfo
    BinPath   map[string]string
    ConfPath  string
    CronRtime int64
}

func GetConfPath() string {
    pkgFile, _ := exec.LookPath(os.Args[0])
    path, _ := filepath.Abs(pkgFile)
    index := strings.LastIndex(path, string(os.PathSeparator))
    return path[:index]
}

func GetConf() (Conf Config, err error) {
    path := GetConfPath()
    Conf, err = GetConfInfo(path)
    if err != nil {
        return
    }
    Conf.ConfPath = path
    cron, cronRtime, err := GetCronList(path)
    if err != nil {
        return
    }
    Conf.Cron = cron
    Conf.CronRtime = cronRtime
    return
}

func GetConfInfo(path string) (conf Config, err error) {
    if path == "" {
        path = GetConfPath()
    }
    file, osErr := os.Open(path + "/Conf/conf.json")
    // 打开文件
    //file, osErr := os.Open("../Conf/conf.json")
    // 关闭文件
    defer file.Close()
    if osErr != nil {
        err = errors.New("读取Conf配置文件错误")
        return
    }
    var tmpConf Config
    //NewDecoder创建一个从file读取并解码json对象的*Decoder，解码器有自己的缓冲，并可能超前读取部分json数据。
    decoder := json.NewDecoder(file)
    //Decode从输入流读取下一个json编码值并保存在v指向的值里
    errJson := decoder.Decode(&tmpConf)
    if errJson != nil {
        err = errors.New("读取Conf配置错误")
        return
    }
    conf = tmpConf
    return
}

func GetCronList(path string) (cron []crontab, cronRtime int64, err error) {
    if path == "" {
        path = GetConfPath()
    }
    fileStat, err := os.Stat(path + "/Conf/cron.json")
    if err != nil {
        return
    }
    cronRtime = fileStat.ModTime().Unix()
    //打开文件
    file, osErr := os.Open(path + "/Conf/cron.json")
    // 关闭文件
    defer file.Close()
    if osErr != nil {
        err = errors.New("读取Cron配置文件错误")
        return
    }
    var tmpCron []crontab
    //NewDecoder创建一个从file读取并解码json对象的*Decoder，解码器有自己的缓冲，并可能超前读取部分json数据。
    decoder := json.NewDecoder(file)
    //Decode从输入流读取下一个json编码值并保存在v指向的值里
    errJson := decoder.Decode(&tmpCron)
    if errJson != nil {
        err = errors.New("读取Cron配置错误")
        return
    }
    cron = tmpCron
    return
}

func CheckCron(path string, lastTime int64) (stat bool, cron []crontab, cronRtime int64, err error) {
    stat = false
    fileStat, err := os.Stat(path + "/Conf/cron.json")
    if err != nil {
        return
    }
    if fileStat.ModTime().Unix() > lastTime {
        stat = true
        cron, cronRtime, err = GetCronList(path)
        if err != nil {
            return
        }
    }
    return
}

//func main() {
//	conf, err := GetConf()
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//	fmt.Println(conf.Db.DbPort)
//}

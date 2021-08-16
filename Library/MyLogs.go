package Library

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"
)

type MyLogs struct {
	Dir           string
	Filename      string
	Filepath      string
	LogExt        string
	Maxlogfilenum int
	FileRootPath  string
	Mode          string
	LogChan       chan string
	HttpReq       *http.Request
}

func NewMyLogs(rootpath string, dir string, logext string, httpreq *http.Request) (myLog *MyLogs, err error) { //, maxlogfilenum int, maxlogfilesize int
	if rootpath == "" {
		err = errors.New("日志根路径不能为空")
		return
	}
	if dir == "" {
		err = errors.New("日志路径不能为空")
		return
	}
	//if maxlogfilenum <= 0 {
	//    maxlogfilenum = 0
	//}
	//if maxlogfilesize <= 0 {
	//    maxlogfilesize = 5000000
	//}
	if logext == "" {
		logext = ".log"
	}
	myLog = &MyLogs{
		Dir:          dir,
		Mode:         "Mylogs",
		Filename:     "",
		LogExt:       logext,
		FileRootPath: rootpath,
		HttpReq:      httpreq,
	}
	myLog.Filepath = rootpath + myLog.GetSeparator() + dir + myLog.GetSeparator() + strconv.Itoa(time.Now().Year()) + strconv.Itoa(int(time.Now().Month()))
	err = myLog.start()
	return
}

/**
 * 开通协程和通道单线程处理写日志
 */
func (m *MyLogs) start() (err error) {
	logChan := make(chan string)
	m.LogChan = logChan
	go func() {
		m.checkFile()
		//尝试打开或创建文件
		file, err := os.OpenFile(m.Filepath+m.GetSeparator()+m.Filename+m.LogExt, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		for x := range logChan {
			check := m.checkFile()
			if check {
				file.Close()
				//文件名变了  尝试打开或创建新文件
				file, err = os.OpenFile(m.Filepath+m.GetSeparator()+m.Filename+m.LogExt, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
			}
			//fmt.Printf(x)
			//如果打开文件错误则不写日志
			if err == nil {
				file.WriteString(x)
			}
		}
		fmt.Println("MyLogs日志通道已关闭")
		file.Close()
	}()
	return
}

/**
 * 关闭通道
 */
func (m *MyLogs) stop() {
	i := 0
	fmt.Println("准备关闭MyLogs日志通道")
	for {
		if len(m.LogChan) < 1 {
			break
		}
		if i > 20 {
			break
		}
		time.Sleep(time.Millisecond * 500)
		i++
	}
	close(m.LogChan)
}

/**
 * 写日志
 */
func (m MyLogs) DoLogs(msg string, priority string, mode string, data interface{}) {
	m.setMode(mode)
	//if m_InitOk == false{
	//    return
	//}
	m.writeLogs(msg, data, priority)
}

/**
 * 写日志到文件
 */
func (m MyLogs) writeLogs(log string, data interface{}, priority string) (err error) {
	errStr := m.getLogType(priority)
	dateStr := "[#" + m.Mode + "#][" + time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05") + "][-" + strconv.FormatInt(time.Now().Unix(), 10) + "-]"
	txtData := ""
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		txtData = " Data:[" + string(jsonData) + "]"
	}
	txtData += "\n"
	str := dateStr + errStr + log + txtData
	m.LogChan <- str
	return
}

/**
 * 写日志
 */
func (m *MyLogs) setMode(mode string) {
	m.Mode = mode
}

/**
 * @abstract 设置日志类型
 * @param String $error 类型
 */
func (m MyLogs) getLogType(errType string) (errStr string) {
	logType := ""
	switch errType {
	case "f":
		logType = "FATAL"
	case "e":
		logType = "ERROR"
	case "n":
		logType = "NOTICE"
	case "d":
		logType = "DEBUG"
	case "w":
		logType = "WARNING"
	default:
		logType = "INFO"
	}
	backtrace := m.runFuncName(4)
	errStr = " [" + logType + "]"
	if errType == "f" || errType == "e" {
		errStr += " [FuncName:" + backtrace["funcname"] + "]"
		if m.HttpReq != nil {
			errStr += " [ip:" + Getip(m.HttpReq) + "] [os:" + GetClientOs(m.HttpReq) + "] "
		}
	}
	errStr += "[Line:" + backtrace["line"] + "] "
	errStr += "[File:" + backtrace["file"] + "] "
	return
}

/**
 * 检测日志文件和路径
 */
func (m *MyLogs) checkFile() (f bool) {
	_ = os.MkdirAll(m.Filepath, 0777)
	day := strconv.Itoa(time.Now().Day())
	if m.Filename == "" {
		m.Filename = fmt.Sprintf("%02s", day)
		f = false
	} else if m.Filename != day {
		m.Filename = fmt.Sprintf("%02s", day)
		f = true
	}
	return
}

/**
 * 获取路径分隔符号
 */
func (m MyLogs) GetSeparator() (separator string) {
	if runtime.GOOS == "windows" {
		separator = "\\"
	} else {
		separator = "/"
	}
	return
}

// 获取正在运行的函数名
func (m MyLogs) runFuncName(skip int) map[string]string {
	pc, file, line, ok := runtime.Caller(skip)
	backtrace := map[string]string{
		"funcname": "",
		"line":     "",
		"file":     "",
	}
	if ok {
		f := runtime.FuncForPC(pc)
		backtrace["funcname"] = f.Name()
		backtrace["line"] = strconv.Itoa(line)
		backtrace["file"] = file
	}

	return backtrace
}

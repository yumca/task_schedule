package Library

import (
    "github.com/thinkeridea/go-extend/exnet"
    "net/http"
    "strings"
)

/**
 * 去空
 */
func TrimEmpty(a []string) (ret []string) {
    aLen := len(a)
    for i := 0; i < aLen; i++ {
        if len(a[i]) == 0 {
            continue
        }
        ret = append(ret, a[i])
    }
    return
}

/**
 * 日志
 */
func DoMyLogs(msg string, types string, mode string, data string) {
    //MyLogs.NewMyLogs()
    //$trace = debug_backtrace(DEBUG_BACKTRACE_IGNORE_ARGS, 1)
    //$this- > logs- > doLog($msg, $data, $type, current($trace))
}

/**
 * 获取ip
 */
func Getip(r *http.Request) string {
    ip := exnet.ClientPublicIP(r)
    if ip == "" {
        ip = exnet.ClientIP(r)
    }
    if ip == "" {
        ip = "0.0.0.0"
    }
    return ip
}

func GetClientOs(r *http.Request) string {
    os := "other"
    userAgent := strings.ToLower(r.Header.Get("User-Agent"))
    if re := strings.IndexAny(userAgent, "iphone"); re != -1 {
        os = "iphone"
    } else if re := strings.IndexAny(userAgent, "android"); re != -1 {
        os = "android"
    } else if re := strings.IndexAny(userAgent, "micromessenger"); re != -1 {
        os = "weixin"
    } else if re := strings.IndexAny(userAgent, "ipad"); re != -1 {
        os = "ipad"
    } else if re := strings.IndexAny(userAgent, "ipod"); re != -1 {
        os = "ipod"
    } else if re := strings.IndexAny(userAgent, "windows nt"); re != -1 {
        os = "pc"
    }
    return os
}

//func GetItemId() string {
//    $hour = date("z") * 24 + date("H");
//    $hour = str_repeat("0", 4 - strlen($hour)) . $hour;
//    //	echo date("y") . $hour . PHP_EOL;
//    return date("y") . $hour . getRandNumber(10);
//}

func GetItemId() string {
    return ""
}

////返回秒随机号
//function getMillisecond() {
//    return (string) time() . (string) mt_rand(10000000, 99999999);
//}

func GetMillisecond() string {
    return ""
}

//
///**
// * 生成固定长度的随机数
// *
// * @param int $length
// * @return string
// */
//function getRandNumber($length = 6) {
//    $num = "";
//    if ($length >= 10) {
//        $t = intval($length / 9);
//        $tail = $length % 9;
//        for ($i = 1; $i <= $t; $i ++) {
//            $num .= substr(mt_rand("1" . str_repeat("0", 9), str_repeat("9", 10)), 1);
//        }
//        $num .= substr(mt_rand("1" . str_repeat("0", $tail), str_repeat("9", $tail + 1)), 1);
//        return $num;
//    } else {
//        return substr(mt_rand("1" . str_repeat("0", $length), str_repeat("9", $length + 1)), 1);
//    }
//}
//
///**
// * ws返回格式化
// */
//function ws_return($signal, $code = 1, $msg = "succ", $data = []) {
//    return [
//        "signal" => $signal,
//        "code" => intval($code),
//        "msg" => $msg,
//        "serverTime" => time(),
//        "data" => $data
//    ];
//}

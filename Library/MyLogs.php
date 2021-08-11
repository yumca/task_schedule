<?php

class MyLogs {

    private $FilePath;
    private $FileName;
    private $m_MaxLogFileNum;
    private $m_RotaType;
    private $m_RotaParam;
    private $m_InitOk;
    private $m_LogCount;
    private $root = '/data/weblog';
    private $logExt = '.log';
    private $Store = 10;
    private $LogLevel = 0;
    private $Mode = '';
    private $email = false;
    //监控目录
    private $MonitorFilePath;
    private $MonitorFileName = 'error';

    /**
     * @abstract 初始化
     * @param String $dir 文件路径
     * @param String $filename 文件名
     * @return 
     */
    function __construct($dir, $store = 10, $filename = '', $maxlogfilenum = 3, $rotatype = 1, $rotaparam = 5000000) {
        if (!defined('PATH')) {
            echo 'PATH配置丢失!';
            exit;
        }
        $this->MonitorFilePath = '/data/weblog'; //监控目录
        if (!empty($filename)) {
            $dot_offset = strpos($filename, ".");
            if ($dot_offset !== false) {
                $this->FileName = substr($filename, 0, $dot_offset);
            } else {
                $this->FileName = $filename;
            }
        } else {
            $this->FileName = date('d');
        }
        $this->Mode = 'task_schedule/' . $dir;
        $dir = str_replace('/', '_', $dir);
        $this->Store = $store;
        if ($store == 20) {
            return;   //如果只保持到数据库则不生成文件夹
        }
        //$this->FilePath = PATH . $this->root . $dir . '/' . date('Ym') . '/';
        $this->FilePath = $this->root . DIRECTORY_SEPARATOR . 'logs' . DIRECTORY_SEPARATOR . 'task_schedule' . DIRECTORY_SEPARATOR . $dir . DIRECTORY_SEPARATOR . date('Ym') . DIRECTORY_SEPARATOR;
        $this->m_MaxLogFileNum = intval($maxlogfilenum);
        $this->m_RotaParam = intval($rotaparam);
        $this->m_RotaType = intval($rotatype);
        $this->m_LogCount = 0;

        $this->m_InitOk = $this->InitDir();
        umask(0000);
        $path = $this->createPath($this->FilePath, $this->FileName);
        if (!$this->isExist($path)) {
            if (!$this->createDir($this->FilePath)) {
                #echo("创建目录失败!");
            }
            if (!$this->createLogFile($path)) {
                #echo("创建文件失败!");
            }
        }
    }

    private function createPath($path = '', $fileName = '') {
        return $path . $fileName . $this->logExt;
    }

    private function InitDir() {
        if (is_dir($this->FilePath) === false) {
            if (!$this->createDir($this->FilePath)) {
                //echo("创建目录失败!");
                //throw exception
                return false;
            }
        }
        return true;
    }

    /**
     * @abstract 设置日志类型
     * @param String $error 类型
     */
    private function setLogType($error = '', $isMonitor = false) {
        switch ($error) {
            case 'f':
                $type = 'FATAL';
                $this->LogLevel = 50;
                //当错误级别为f,设置为30;
                $this->Store = 30;
                break;
            case 'e':
                $type = 'ERROR';
                $this->LogLevel = 10;
                //当错误级别为e,设置为30;
                $this->Store = 30;
                break;
            case 'n':
                $type = 'NOTICE';
                $this->LogLevel = 20;
                break;
            case 'd':
                $type = 'DEBUG';
                $this->LogLevel = 30;
                break;
            case 'w':
                $type = 'WARNING';
                $this->LogLevel = 40;
                //当错误级别为w,设置为30;
                $this->Store = 30;
                break;
            default;
                $type = 'INFO';
                break;
        }
        $str = ' [' . $type . '] ';
        if (in_array($error, array('e', 'f'))) {
            if ($isMonitor) {
                $str .= '[' . $this->Mode . '] ';
            }
            $str .= '[path:' . $_SERVER["SCRIPT_FILENAME"] . '] [ip:' . $this->getip() . '] [os:' . $this->get_client_os() . '] ';
        }
        return $str;
    }

    /**
     * @abstract 写入日志
     * @param String $log 内容
     * @param Array $data 数据
     */
    public function doLog($log = '', $data = array(), $priority = '', $backtrace = '') {
        if (empty($log)) {
            return false;
        }
        $logType = $this->setLogType($priority);
        if ($this->Store == 10 || $this->Store == 30) {
            if ($this->m_InitOk == false)
                return false;
            $path = $this->getLogFilePath($this->FilePath, $this->FileName) . $this->logExt;
            $handle = @fopen($path, "a+");
            if ($handle === false) {
                return false;
            }
            $txtData = !empty($data) ? ' Data:[' . json_encode($data, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . ']' : '';
            $datestr = '[#' . $this->Mode . '#][' . strftime("%Y-%m-%d %H:%M:%S") . '][-' . REQUEST_LINE_ID . '-]';
            $line = isset($backtrace['line']) && !empty($backtrace['line']) ? '[Line:' . $backtrace['line'] . '] ' : '';
            $file = isset($backtrace['file']) && !empty($backtrace['file']) ? '[File:' . $backtrace['file'] . '] ' : '';
            //$caller_info = $this->get_caller_info();
            if (!@fwrite($handle, $datestr . $logType . $file . $line . $log . $txtData . ';' . "\n")) {//写日志失败
                echo("写入日志失败");
            }
            @fclose($handle);
            $this->RotaLog();
        }
    }

    /**
     * @abstract 写入监控日志目录
     * @param String $log 内容
     */
    private function doMonitorLog($log = '', $data = array(), $priority) {
        //判断文件夹是否存在
        if (!is_dir($this->MonitorFilePath)) {
            return false;
        }
        $logType = $this->setLogType($priority, true);
        $path = $this->getLogFilePath($this->MonitorFilePath, $this->MonitorFileName) . $this->logExt;

        $handle = @fopen($path, "a+");
        if ($handle === false) {
            return;
        }
        $txtData = !empty($data) ? ' Data:[' . json_encode($data, JSON_UNESCAPED_UNICODE) . ']' : '';
        $datestr = '[' . strftime("%Y-%m-%d %H:%M:%S") . ']';
        if (!@fwrite($handle, $datestr . $logType . $log . $txtData . ';' . "\n")) {//写日志失败
            echo("写入日志失败");
        }
        @fclose($handle);
    }

    private function get_caller_info() {
        $ret = debug_backtrace();
        foreach ($ret as $item) {
            if (isset($item['class']) && 'Logs' == $item['class']) {
                continue;
            }
            $file_name = basename($item['file']);
            return <<<S
          {$file_name}:{$item['line']}
S;
        }
    }

    private function RotaLog() {
        $file_path = $this->getLogFilePath($this->FilePath, $this->FileName) . $this->logExt;
        if ($this->m_LogCount % 10 == 0)
            clearstatcache();
        ++$this->m_LogCount;
        $file_stat_info = stat($file_path);
        if ($file_stat_info === FALSE)
            return;
        if ($this->m_RotaType != 1)
            return;

        //echo "file: ".$file_path." vs ".$this->m_RotaParam."\n";
        if ($file_stat_info['size'] < $this->m_RotaParam)
            return;

        $raw_file_path = $this->getLogFilePath($this->FilePath, $this->FileName);
        $file_path = $raw_file_path . ($this->m_MaxLogFileNum - 1) . $this->logExt;
        //echo "lastest file:".$file_path."\n";
        if ($this->isExist($file_path)) {
            unlink($file_path);
        }
        for ($i = $this->m_MaxLogFileNum - 2; $i >= 0; $i--) {
            if ($i == 0) {
                $file_path = $raw_file_path . $this->logExt;
            } else {
                $file_path = $raw_file_path . $i . $this->logExt;
            }
            if ($this->isExist($file_path)) {
                $new_file_path = $raw_file_path . ($i + 1) . $this->logExt;
                if (rename($file_path, $new_file_path) < 0) {
                    continue;
                }
            }
        }
    }

    private function isExist($path) {
        return file_exists($path);
    }

    // 推送日志到队列
    public function logsToQueue($name, $data) {
        exit;
        //初始化队列推送类
        $send = new SendQueue(C('MQ_CONN_ARGS'));
        // 设置队列名称
        $send->setSendOption($name, AMQP_DURABLE);
        // 推送数据
        $ret = $send->send($data);
        if ($ret) {
            return true;
        } else {
            return false;
        }
    }

    /**
     * @abstract 创建目录
     * @param <type> $dir 目录名
     * @return bool
     */
    private function createDir($dir) {
        return is_dir($dir) or ( $this->createDir(dirname($dir)) and @ mkdir($dir, 0777) and chmod($dir, 0777) and @ chmod($dir, 0777));
    }

    /**
     * @abstract 创建日志文件
     * @param String $path
     * @return bool
     */
    private function createLogFile($path) {
        $handle = @fopen($path, "w"); //创建文件
        @fclose($handle);
        chmod($path, 0777);
        return $this->isExist($path);
    }

    /**
     * @abstract 创建路径
     * @param String $dir 目录名
     * @param String $filename 
     */
    private function getLogFilePath($dir, $filename) {
        return $dir . "/" . $filename;
    }

    //获取访问者的操作系统
    private function get_client_os() {
        $os = 'other';
        $userAgent = isset($_SERVER["HTTP_USER_AGENT"]) ? strtolower($_SERVER["HTTP_USER_AGENT"]) : '';
        if ($re = strripos($userAgent, 'iphone')) {
            $os = 'iphone';
        } else if ($re = strripos($userAgent, 'android')) {
            $os = 'android';
        } else if ($re = strripos($userAgent, 'micromessenger')) {
            $os = 'weixin';
        } else if ($re = strripos($userAgent, 'ipad')) {
            $os = 'ipad';
        } else if ($re = strripos($userAgent, 'ipod')) {
            $os = 'ipod';
        } else if ($re = strripos($userAgent, 'windows nt')) {
            $os = 'pc';
        }
        return $os;
    }

    private function getip() {
        $ip = '0.0.0.0';
        if (!empty($_SERVER['HTTP_CLIENT_IP'])) {
            $ip = $_SERVER['HTTP_CLIENT_IP'];
        }
        if (!empty($_SERVER['HTTP_X_FORWARDED_FOR'])) {
            $ips = explode(', ', $_SERVER['HTTP_X_FORWARDED_FOR']);
            if ($ip) {
                array_unshift($ips, $ip);
                $ip = FALSE;
            }
            for ($i = 0; $i < count($ips); $i++) {
                if (!eregi('^(10│172.16│192.168).', $ips[$i])) {
                    $ip = $ips[$i];
                    break;
                }
            }
        }
        if (!empty($_SERVER['REMOTE_ADDR'])) {
            $ip = $_SERVER['REMOTE_ADDR'];
        }
        return $ip;
    }

}

?>
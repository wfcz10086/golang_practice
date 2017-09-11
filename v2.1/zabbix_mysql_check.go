package main
//导入相关库

import (
	"flag"
	"fmt"
	"log"
	"os"
	"encoding/json"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"
)
//定义命令行参数解析flag
var (
	host              = flag.String("host", "127.0.0.1", "`MySQL的服务器默认为127.0.0.1 ")
//定义主机
	user              = flag.String("user", "root", "MySQL 默认为root")
//定义账号，默认root
	pass              = flag.String("pass", "", "默认为空")
//定义密码，默认为空
	port              = flag.String("port", "3306", "MySQL 默认端口为3306")
//定义端口，默认为3306
	debugLog          = flag.String("debug_log", "", "不加此项 默认是不打印调试日志的")
//定义调试日志日志
	cacheDir          = flag.String("cache_dir", "/tmp", "日志输出目录为/tmp")
//定义日志存储位置
	nocache           = flag.Bool("nocache", false, "默认是不打开的")
//定义是否开启缓存

	pollTime          = flag.Int("poll_time", 30, "调整轮训时间")
	discoveryPort     = flag.Bool("discovery_port", false, "自动发现数据库多实例，默认不打开，如果调用，那么debug参数将无效")
//定义自动发现数据库端口
	useSudo           = flag.Bool("sudo", true, "Use `sudo netstat...`")

	version           = flag.Bool("version", false, "版本号")

	// log
	debugLogFile *os.File

	//regexps
	regSpaces     = regexp.MustCompile("\\s+")
	regNumbers    = regexp.MustCompile("\\d+")


	Version string
)


func main() {
	flag.Parse()
//处理version的逻辑
	if *version {
		fmt.Println("version: 1.00")
		os.Exit(1)
	}

	//处理debug日志的逻辑
	{
		if *debugLog != "" {
		//为空处理
			var err error
			debugLogFile, err = os.OpenFile(*debugLog, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
			//设定文件打开三种方式 读写，追加 ，文件不存在创建 
			if nil != err {
			//打开文件错误处理
				fmt.Printf("Can not create file using (%s as path), err: %s", *debugLog, err)
			}
			defer debugLogFile.Close()
			//关闭文件处理
			debugLogFile.Truncate(0)
			//重置文件大小为0
		}
		log.SetOutput(debugLogFile)
		//设置日志输出
	}
	//log.Fatalf("您执行了查询")
	
	//自动发现myslq多实例
	if *discoveryPort {
		discoveryMysqldPort()
		log.Fatalln("discovery Mysqld Port done, exit.")
	}


	//处理账号和密码逻辑还有目标主机
	
	

	{
		if *user == "" || *pass == "" {
			fmt.Fprintln(os.Stderr, "Please set mysql user and password use --user= --pass= !")
			log.Fatalln("flag user or pass is empty, exit !")
		}
		*host = strings.Replace(*host, ":", "", -1)
		*host = strings.Replace(*host, "/", "_", -1)
	}

	// 设定生成日志方式
	var cacheFile *os.File
	{
		cacheFilePath := *cacheDir + "/actiontech_" + *host + "_" + *port + "-mysql_zabbix_stats.txt"
		//设定日志生成方式 /tmp_actiontech_主机_端口_-mysql_zabbix_stats.txt
		log.Printf("cacheFilePath = %s\n", cacheFilePath)
		if !(*nocache) {
			var err error
			cacheFile, err = checkCache(cacheFilePath)
			if nil != err {
				log.Fatalf("checkCache err:%s\n", err)
			}
			defer cacheFile.Close()
		} else {
			log.Println("Caching is disabled.")
		}
	}

}
//自动发现多实例函数
func discoveryMysqldPort() {
	//创建集合
	data := make([]map[string]string, 0)
	//调用json模块
	enc := json.NewEncoder(os.Stdout)
	cmd := "netstat -ntlp |awk '/mysqld/ {print $4}'|awk -F ':' '{print $NF}'"
	//自动发现端口组装
	if *useSudo {
		cmd = "sudo " + cmd
	}
	//调用sudo
	log.Println("discoveryMysqldPort:find mysql port cmd:", cmd)
	
	out, err := exec.Command("sh", "-c", cmd).Output()
	//执行命令
	log.Println("discoveryMysqldPort:cmd out:", string(out))
	if nil != err {
		log.Println("discoveryMysqldPort err:", err)
	}
	//打印相关输出
	fields := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	//分割字符
	for _, field := range fields {
	//组装map
		if "" == field {
			continue
		}
		mp := map[string]string{
			"{#MYSQLPORT}": field,
		}
		data = append(data, mp)
	}

	formatData := map[string][]map[string]string{
		"data": data,
	}
	enc.Encode(formatData)
	//map转json
}


//设置日志输出格式
func checkCache(filepath string) (*os.File, error) {
	fp, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE, 0600)
	//设定打开方式
	if nil != err {
		return nil, err
	}
	//设定文件锁
	err = syscall.Flock(int(fp.Fd()), syscall.LOCK_SH)
	if nil != err {
		return nil, err
	}

	fileinfo, err := fp.Stat()
	if nil != err {
		return nil, err
	}

	log.Println("fileinfo.Modtime:", fileinfo.ModTime().Unix())
	log.Println("time.Mow.Unix", time.Now().Unix())
	log.Println("fileinfo.Size", fileinfo.Size())

	if fileinfo.Size() > 0 && (fileinfo.ModTime().Unix()+int64(*pollTime/2)) > time.Now().Unix() {
		log.Println("Using the cache file")
		return nil, nil
	} else {
		log.Println("The cache file seems too small or stale")
	
		err = syscall.Flock(int(fp.Fd()), syscall.LOCK_EX)
		if nil != err {
			return nil, err
		}


		if fileinfo.Size() > 0 && (fileinfo.ModTime().Unix()+int64(*pollTime/2)) > time.Now().Unix() {
			log.Println("Using the cache file")
			return nil, nil
		}
		fp.Truncate(0)
	}
	return fp, nil

}

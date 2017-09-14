package main
//导入相关库

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"
	"strconv"
)

const (
	SHOW_STATUS = iota
	SHOW_VARIABLES
	SHOW_SLAVE_STATUS
	SHOW_PROCESSLIST
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
	procs             = flag.Bool("procs", true, "查看Mysql进程参数 雷同SHOW PROCESSLIST")
//定义查询SHOW PROCESSLIST参数
	slave             = flag.Bool("slave", true, "查看Mysql进程参数")
//定义查询slave 状态
//是否使用sudo权限
	items             = flag.String("items", "", "检查项目")
//定义items

	heartbeat         = flag.Bool("heartbeat", false, "同步延迟计算")
	heartbeatUtc      = flag.Bool("heartbeat_utc", false, "延时 时间表达使用UTC格式")
	heartbeatServerId = flag.String("heartbeat_server_id", "0", "同步Server id，如果不设置默认为0")
	heartbeatTable    = flag.String("heartbeat_table", "percona.heartbeat", "`db.tbl`.")
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

	//检查端口占用
	if *items == "mysqld_port_listen" {
		// default port: 3306
		if verifyMysqldPort(*port) {
			fmt.Println(*port, ":", 1)
		} else {
			fmt.Println(*port, ":", 0)
		}
		log.Fatalln("verify Mysqld Port done, exit.")
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

	collectionExist, collectionInfo := collect()
	result := parse(collectionExist, collectionInfo)
	new(result, cacheFile)


}

func collect() ([]bool, []map[string]string) {
	var db *sql.DB
		var err error
		log.Printf("Connecting mysql: user %s, pass %s, host %s, port %s", *user, *pass, *host, *port)
		db, err = sql.Open("mysql", *user+":"+*pass+"@tcp("+*host+":"+*port+")/")
		if nil != err {
			log.Fatalf("sql.Open err:(%s) exit !", err)
		}
		defer db.Close()
		if err := db.Ping(); nil != err {
			log.Fatalf("db.Ping err:(%s) exit !", err)
		}

	collectionInfo := make([]map[string]string, 8)
	collectionExist := []bool{true, true, false, false, false, false, false, false}

	collectionInfo[SHOW_STATUS] = collectAllRowsToMap("variable_name", "value", db, "SHOW /*!50002 GLOBAL */ STATUS")
	collectionInfo[SHOW_VARIABLES] = collectAllRowsToMap("variable_name", "value", db, "SHOW VARIABLES")
////////////////////////////////////////////2017年9月14日15:03:11
//匹配slave相关模块 
if *slave {
		collectionExist[SHOW_SLAVE_STATUS] = true
		collectionInfo[SHOW_SLAVE_STATUS] = make(map[string]string)

		queryResult := tryQueryIfAvailable(db, "SHOW SLAVE STATUS NONBLOCKING", "SHOW SLAVE STATUS NOLOCK", "SHOW SLAVE STATUS")
		log.Println("get slave status: ", queryResult)
		if 0 == len(queryResult) {
			log.Println("show slave empty, assume it is a master")
		} else {
			queryResult[0] = changeKeyCase(queryResult[0])
			if 1 < len(queryResult) {
		
				log.Println("show slave multi rows, assume it is a multi source replication")

				var maxLag int64 = -1
				var totalRelayLogSpace int64 = 0
				for i, resultMap := range queryResult {
					resultMap = changeKeyCase(resultMap)
					if resultMap["slave_io_running"] != "Yes" {
						log.Printf("%dth row slave_io_running != Yes", i)
						queryResult[0]["slave_io_running"] = "No"
					}
					if resultMap["slave_sql_running"] != "Yes" {
						log.Printf("%dth row slave_sql_running != Yes", i)
						queryResult[0]["slave_sql_running"] = "No"
					}

					// 
					if resultMap["seconds_behind_master"] != "NULL" && convStrToInt64(resultMap["seconds_behind_master"]) > maxLag {
						log.Printf("%dth row seconds_behind_master may be max", i)
						maxLag = convStrToInt64(resultMap["seconds_behind_master"])
						queryResult[0]["seconds_behind_master"] = resultMap["seconds_behind_master"]
					}

					//获取同步空间
					log.Printf("%dth row relay_log_space is %s", i, resultMap["relay_log_space"])
					totalRelayLogSpace += convStrToInt64(resultMap["relay_log_space"])
				}
				queryResult[0]["relay_log_space"] = strconv.FormatInt(totalRelayLogSpace, 10)
			}
			stringMapAdd(collectionInfo[SHOW_SLAVE_STATUS], queryResult[0])

			stringMapAdd(collectionInfo[SHOW_SLAVE_STATUS], getDelayFromHeartbeat(db))
		}
		log.Println("collectionInfo slave: ", collectionInfo[SHOW_SLAVE_STATUS])
	}
///////////////////////////////////////////
	
	if *procs {
		collectionExist[SHOW_PROCESSLIST] = true
		collectionInfo[SHOW_PROCESSLIST] = AllRowsAsMapValue("show_processlist_", "state", db, "SHOW PROCESSLIST")
		log.Println("collectionInfo show processlist:", collectionInfo[SHOW_PROCESSLIST])
	}

	return collectionExist, collectionInfo
}

func parse(collectionExist []bool, collectionInfo []map[string]string) map[string]string {
	stat := make(map[string]string)
	stringMapAdd(stat, collectionInfo[SHOW_STATUS])
	stringMapAdd(stat, collectionInfo[SHOW_VARIABLES])
	
	{
		//设定进程相关map
		procsStateMap := map[string]int64{
			"State_closing_tables":       0,
			"State_copying_to_tmp_table": 0,
			"State_end":                  0,
			"State_freeing_items":        0,
			"State_init":                 0,
			"State_locked":               0,
			"State_login":                0,
			"State_preparing":            0,
			"State_reading_from_net":     0,
			"State_sending_data":         0,
			"State_sorting_result":       0,
			"State_statistics":           0,
			"State_updating":             0,
			"State_writing_to_net":       0,
			"State_none":                 0,
			"State_other":                0,
		}
		//进程相关查询逻辑
		if collectionExist[SHOW_PROCESSLIST] {
			var state string
			reg := regexp.MustCompile("^(Table lock|Waiting for .*lock)$")
			for _, value := range collectionInfo[SHOW_PROCESSLIST] {
				if value == "" {
					value = "none"
				}

				state = reg.ReplaceAllString(value, "locked")
				state = strings.Replace(strings.ToLower(value), " ", "_", -1)
				if _, ok := procsStateMap["State_"+state]; ok {
					procsStateMap["State_"+state]++
				} else {
					procsStateMap["State_other"]++
				}
			}
		}
		intMapAdd(stat, procsStateMap)
	}

	//////////////////////////////////////////
	if collectionExist[SHOW_SLAVE_STATUS] {
		if 0 == len(collectionInfo[SHOW_SLAVE_STATUS]) {
			log.Println("it is a master!set running_slave = 1")
			stat["running_slave"] = "1"
			stat["slave_lag"] = "0"
		} else if collectionInfo[SHOW_SLAVE_STATUS]["slave_io_running"] == "Yes" && collectionInfo[SHOW_SLAVE_STATUS]["slave_sql_running"] == "Yes" {
			stat["running_slave"] = "1"
		} else {
			stat["running_slave"] = "0"
		}

		if v, ok := collectionInfo[SHOW_SLAVE_STATUS]["relay_log_space"]; ok {
			stat["relay_log_space"] = v
		}

		if v, ok := collectionInfo[SHOW_SLAVE_STATUS]["seconds_behind_master"]; ok {
			stat["slave_lag"] = v
		}

		if v, ok := collectionInfo[SHOW_SLAVE_STATUS]["delay_from_heartbeat"]; ok {
			stat["slave_lag"] = v
		}

		if collectionInfo[SHOW_SLAVE_STATUS]["slave_sql_running"] == "Yes" {
			stat["slave_running"] = stat["slave_lag"]
			stat["slave_stopped"] = "0"
		} else {
			stat["slave_running"] = "0"
			stat["slave_stopped"] = stat["slave_lag"]
		}
	}

	/////////////////////////////////////////
	return stat

}
//查询值和集合处理
func FirstRowAsMapValue(key string, valueColName string, db *sql.DB, querys ...string) map[string]string {
	result := make(map[string]string)
	queryResult := tryQueryIfAvailable(db, querys...)
	if 0 == len(queryResult) {
		log.Println("collectFirstRowAsMapValue:Got nothing from query: ", querys)
		return result
	}
	mp := changeKeyCase(queryResult[0])
	if _, ok := mp[valueColName]; !ok {
		log.Printf("collectFirstRowAsMapValue:Couldn't get %s from %s\n", valueColName, querys)
		return result
	}
	result[key] = mp[valueColName]
	return result
}

//map添加到最后的输出结果
func intMapAdd(m1 map[string]string, m2 map[string]int64) {
	for k, v := range m2 {
		if _, ok := m1[k]; ok {
			log.Fatal("key conflict:", k)
		}
		m1[k] = strconv.FormatInt(v, 10)
	}
}
//总查询函数
func query(db *sql.DB, query string) []map[string]string {
	log.Println("exec query:", query)
	result := make([]map[string]string, 0, 500)

	rows, err := db.Query(query)
	if nil != err {
		log.Println("db.Query err:", err)
		return result
	}
	defer func(rows *sql.Rows) {
		if rows != nil {
			rows.Close()
		}
	}(rows)

	columnsName, err := rows.Columns()
	if nil != err {
		log.Println("rows.Columns err:", err)
		return result
	}

	
	values := make([]sql.RawBytes, len(columnsName))



	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	for rows.Next() {

		err = rows.Scan(scanArgs...)
		if nil != err {
			log.Println("rows.Next err:", err)
		}


		row_map := make(map[string]string)
		for i, col := range values {
			if col == nil {
				row_map[columnsName[i]] = "NULL"
			} else {
				row_map[columnsName[i]] = string(col)
			}
		}
		result = append(result, row_map)
	}

	err = rows.Err()
	if nil != err {
		log.Println("rows.Err:", err)
	}
	return result
}
//MYSQL列值转集合
func AllRowsAsMapValue(preKey string, valueColName string, db *sql.DB, querys ...string) map[string]string {
	result := make(map[string]string)
	for i, mp := range tryQueryIfAvailable(db, querys...) {
		mp = changeKeyCase(mp)
		if _, ok := mp[valueColName]; !ok {
			log.Printf("AllRowsAsMapValue:Couldn't get %s from %s\n", valueColName, querys)
			return result
		}
		result[preKey+strconv.Itoa(i)] = mp[valueColName]
	}
	return result
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
//检验数据库实例端口是否正确
func verifyMysqldPort(port string) bool {
	//netstat命令组装
	cmd := "netstat -ntlp |awk -F '[ ]+|/' '$4~/:" + port + "$/{print $8}'"
	//调用sudo
	if *useSudo {
		cmd = "sudo " + cmd
	}
	log.Println("verifyMysqldPort:find port Program name cmd:", cmd)
	//执行命令
	out, err := exec.Command("sh", "-c", cmd).Output()
	log.Println("verifyMysqldPort:cmd out:", string(out))
	if nil != err {
		log.Println("verifyMysqldPort err:", err)
	}
	//如果返回mysqld字符
	if string(out) == "mysqld\n" {
		return true
	}
	return false
}


//大小写转换
func changeKeyCase(m map[string]string) map[string]string {
	lowerMap := make(map[string]string)
	for k, v := range m {
		lowerMap[strings.ToLower(k)] = v
	}
	return lowerMap
}
//键值转集合
func collectAllRowsToMap(keyColName string, valueColName string, db *sql.DB, querys ...string) map[string]string {
	result := make(map[string]string)
	for _, mp := range tryQueryIfAvailable(db, querys...) {

		mp = changeKeyCase(mp)
		result[mp[keyColName]] = mp[valueColName]
	}
	return result
}
//键值是否可查询
func tryQueryIfAvailable(db *sql.DB, querys ...string) []map[string]string {
	result := make([]map[string]string, 0, 500)
	for _, q := range querys {
		result = query(db, q)
		if 0 == len(result) {
			log.Println("tryQueryIfAvailable:Got nothing from sql: ", q)
			continue
		}
		return result
	}
	return result
}

//字符 集合转换
func stringMapAdd(m1 map[string]string, m2 map[string]string) {
	for k, v := range m2 {
		if _, ok := m1[k]; ok {
			log.Fatal("key conflict:", k)
		}
		m1[k] = v
	}
}

//获取延迟时间
func getDelayFromHeartbeat(db *sql.DB) map[string]string {
	result := make(map[string]string)
	if !*heartbeat {
		return result
	}
	var nowFunc string
	if *heartbeatUtc {
		nowFunc = "UNIX_TIMESTAMP(UTC_TIMESTAMP)"
	} else {
		nowFunc = "UNIX_TIMESTAMP()"
	}
	query := fmt.Sprintf("SELECT MAX(%s - ROUND(UNIX_TIMESTAMP(ts))) AS delay FROM %s WHERE %s = 0 OR server_id = %s", nowFunc, *heartbeatTable, *heartbeatServerId, *heartbeatServerId)
	result = FirstRowAsMapValue("delay_from_heartbeat", "delay", db, query)
	return result
}


//字符类型转换
func convStrToInt64(s string) int64 {
	log.Println("convStrToInt64: string:", s)
	value := regexp.MustCompile("\\d+").FindString(s)
	i, err := strconv.ParseInt(value, 10, 64)
	if nil != err {
		log.Fatalf("convStrToInt64 err: parse(%v) to int64 err:%v\n", value, err)
	}
	return i
}
//打印总程序
func new(result map[string]string, fp *os.File) {
	key := []string{
		"Key_read_requests",
		"Key_reads",
		"Key_write_requests",
		"Key_writes",
		"history_list",
		"innodb_transactions",
		"read_views",
		"current_transactions",
		"locked_transactions",
		"active_transactions",
		"pool_size",
		"free_pages",
		"database_pages",
		"modified_pages",
		"pages_read",
		"pages_created",
		"pages_written",
		"file_fsyncs",
		"file_reads",
		"file_writes",
		"log_writes",
		"pending_aio_log_ios",
		"pending_aio_sync_ios",
		"pending_buf_pool_flushes",
		"pending_chkp_writes",
		"pending_ibuf_aio_reads",
		"pending_log_flushes",
		"pending_log_writes",
		"pending_normal_aio_reads",
		"pending_normal_aio_writes",
		"ibuf_inserts",
		"ibuf_merged",
		"ibuf_merges",
		"spin_waits",
		"spin_rounds",
		"os_waits",
		"rows_inserted",
		"rows_updated",
		"rows_deleted",
		"rows_read",
		"Table_locks_waited",
		"Table_locks_immediate",
		"Slow_queries",
		"Open_files",
		"Open_tables",
		"Opened_tables",
		"innodb_open_files",
		"open_files_limit",
		"table_cache",
		"Aborted_clients",
		"Aborted_connects",
		"Max_used_connections",
		"Slow_launch_threads",
		"Threads_cached",
		"Threads_connected",
		"Threads_created",
		"Threads_running",
		"max_connections",
		"thread_cache_size",
		"Connections",
		"slave_running",
		"slave_stopped",
		"Slave_retried_transactions",
		"slave_lag",
		"Slave_open_temp_tables",
		"Qcache_free_blocks",
		"Qcache_free_memory",
		"Qcache_hits",
		"Qcache_inserts",
		"Qcache_lowmem_prunes",
		"Qcache_not_cached",
		"Qcache_queries_in_cache",
		"Qcache_total_blocks",
		"query_cache_size",
		"Questions",
		"Com_update",
		"Com_insert",
		"Com_select",
		"Com_delete",
		"Com_replace",
		"Com_load",
		"Com_update_multi",
		"Com_insert_select",
		"Com_delete_multi",
		"Com_replace_select",
		"Select_full_join",
		"Select_full_range_join",
		"Select_range",
		"Select_range_check",
		"Select_scan",
		"Sort_merge_passes",
		"Sort_range",
		"Sort_rows",
		"Sort_scan",
		"Created_tmp_tables",
		"Created_tmp_disk_tables",
		"Created_tmp_files",
		"Bytes_sent",
		"Bytes_received",
		"innodb_log_buffer_size",
		"unflushed_log",
		"log_bytes_flushed",
		"log_bytes_written",
		"relay_log_space",
		"binlog_cache_size",
		"Binlog_cache_disk_use",
		"Binlog_cache_use",
		"binary_log_space",
		"innodb_locked_tables",
		"innodb_lock_structs",
		"State_closing_tables",
		"State_copying_to_tmp_table",
		"State_end",
		"State_freeing_items",
		"State_init",
		"State_locked",
		"State_login",
		"State_preparing",
		"State_reading_from_net",
		"State_sending_data",
		"State_sorting_result",
		"State_statistics",
		"State_updating",
		"State_writing_to_net",
		"State_none",
		"State_other",
		"Handler_commit",
		"Handler_delete",
		"Handler_discover",
		"Handler_prepare",
		"Handler_read_first",
		"Handler_read_key",
		"Handler_read_next",
		"Handler_read_prev",
		"Handler_read_rnd",
		"Handler_read_rnd_next",
		"Handler_rollback",
		"Handler_savepoint",
		"Handler_savepoint_rollback",
		"Handler_update",
		"Handler_write",
		"innodb_tables_in_use",
		"innodb_lock_wait_secs",
		"hash_index_cells_total",
		"hash_index_cells_used",
		"total_mem_alloc",
		"additional_pool_alloc",
		"uncheckpointed_bytes",
		"ibuf_used_cells",
		"ibuf_free_cells",
		"ibuf_cell_count",
		"adaptive_hash_memory",
		"page_hash_memory",
		"dictionary_cache_memory",
		"file_system_memory",
		"lock_system_memory",
		"recovery_system_memory",
		"thread_hash_memory",
		"innodb_sem_waits",
		"innodb_sem_wait_time_ms",
		"Key_buf_bytes_unflushed",
		"Key_buf_bytes_used",
		"key_buffer_size",
		"Innodb_row_lock_time",
		"Innodb_row_lock_waits",
		"Query_time_count_00",
		"Query_time_count_01",
		"Query_time_count_02",
		"Query_time_count_03",
		"Query_time_count_04",
		"Query_time_count_05",
		"Query_time_count_06",
		"Query_time_count_07",
		"Query_time_count_08",
		"Query_time_count_09",
		"Query_time_count_10",
		"Query_time_count_11",
		"Query_time_count_12",
		"Query_time_count_13",
		"Query_time_total_00",
		"Query_time_total_01",
		"Query_time_total_02",
		"Query_time_total_03",
		"Query_time_total_04",
		"Query_time_total_05",
		"Query_time_total_06",
		"Query_time_total_07",
		"Query_time_total_08",
		"Query_time_total_09",
		"Query_time_total_10",
		"Query_time_total_11",
		"Query_time_total_12",
		"Query_time_total_13",
		"wsrep_replicated_bytes",
		"wsrep_received_bytes",
		"wsrep_replicated",
		"wsrep_received",
		"wsrep_local_cert_failures",
		"wsrep_local_bf_aborts",
		"wsrep_local_send_queue",
		"wsrep_local_recv_queue",
		"wsrep_cluster_size",
		"wsrep_cert_deps_distance",
		"wsrep_apply_window",
		"wsrep_commit_window",
		"wsrep_flow_control_paused",
		"wsrep_flow_control_sent",
		"wsrep_flow_control_recv",
		"pool_reads",
		"pool_read_requests",
		"running_slave",
		"query_rt100s",
		"query_rt10s",
		"query_rt1s",
		"query_rt100ms",
		"query_rt10ms",
		"query_rt1ms",
		"query_rtavg",
		"query_avgrt",
	}

	// Return the output.
	output := make(map[string]string)
	for _, v := range key {
		// 如果没有定义值，就返回-1
		// 0最小值，所以它会被视为缺失值
		if val, ok := result[v]; ok {
			output[v] = val
		} else {
			output[v] = "-1"
		}
	}

	if fp != nil {
		log.Println("write file:")
		enc := json.NewEncoder(fp)
		enc.Encode(output)
		/*		for k, v := range output {
				fmt.Fprintf(fp, "%v:%v ", k, v)
			}*/
	}

	if *items != "" {
		fields := strings.Split(*items, ",")
		for _, field := range fields {
			if val, ok := output[field]; ok {
				fmt.Println(field, ":", val)
			} else {
				fmt.Println(field + " do not exist")
			}
		}

	}

}


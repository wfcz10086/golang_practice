//数据库连接池测试
package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net/http"
)

var db *sql.DB

func init() {
//初始化
	db, _ = sql.Open("mysql", "root:dnt@tcp(127.0.0.1:3306)/test?charset=utf8")
//设置数据池打开和句柄参数
	db.SetMaxOpenConns(2000)
	db.SetMaxIdleConns(1000)
	db.Ping()
}

func main() {
	startHttpServer()
//启动函数
}

func startHttpServer() {
	http.HandleFunc("/pool", pool)
//定义路由
	err := http.ListenAndServe(":9090", nil)
//定义端口
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func pool(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT * FROM user limit 1")
	defer rows.Close()
	checkErr(err)

	columns, _ := rows.Columns()
//定义map
	scanArgs := make([]interface{}, len(columns))
	values := make([]interface{}, len(columns))
//循环使用指针
	for j := range values {
		scanArgs[j] = &values[j]
	}

	record := make(map[string]string)
	for rows.Next() {
		//将行数据保存到record字典
		err = rows.Scan(scanArgs...)
		for i, col := range values {
			if col != nil {
				record[columns[i]] = string(col.([]byte))
			}
		}
	}

	fmt.Println(record)
	fmt.Fprintln(w, "finish")
}

func checkErr(err error) {
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}

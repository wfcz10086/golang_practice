package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)
//go get github.com/go-sql-driver/mysql

func main() {
	insert()
}

func insert() {
	db, err := sql.Open("mysql", "root:dnt@/test?charset=utf8")
	checkErr(err)

	stmt, err := db.Prepare(`INSERT user (user_name,user_age,user_sex) values (?,?,?)`)
	checkErr(err)
	res, err := stmt.Exec("tony", 20, 1)
	checkErr(err)
	id, err := res.LastInsertId()
	checkErr(err)
	fmt.Println(id)
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

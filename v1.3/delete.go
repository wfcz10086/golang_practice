package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)
//go get github.com/go-sql-driver/mysql

func main() {
	remove()
}

func remove() {
	db, err := sql.Open("mysql", "root:dnt@/test?charset=utf8")
	checkErr(err)

	stmt, err := db.Prepare(`DELETE FROM user WHERE user_id=?`)
	checkErr(err)
	res, err := stmt.Exec(1)
	checkErr(err)
	num, err := res.RowsAffected()
	checkErr(err)
	fmt.Println(num)
}



func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

package main

import (
	"fmt"
	"log"
	"net/http"
)


var indexHTML = `<html>
<head>
	<meta http-equiv="Content-type" content="text/html; charset=utf-8">
	<title>测试</title>
</head>
<body>
	<form action="/page" method="post">
		用户名：<br>
		<input name="username" type="text"><br>
		请输入文本：<br>
		<textarea name="usertext"></textarea><br>
		<input type="submit" value="提交">
	</form>
</body>
</html>`

// 用于将页面重定向到主页
var redirectHTML = `<html>
<head>
	<meta http-equiv="Content-type" content="text/html; charset=utf-8">
	<meta http-equiv="Refresh" content="0; url={{.}}">
</head>
<body></body>
</html>`

func index(w http.ResponseWriter, r *http.Request) {
	// 向客户端写入我们准备好的页面
	fmt.Fprintf(w, indexHTML)
}



func page(w http.ResponseWriter, r *http.Request) {
	// 我们规定必须通过 POST 提交数据
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
		log.Println(err)
		
		}
	// 获取客户端输入的内容
	userName := r.Form.Get("username")
	userText := r.Form.Get("usertext")
	// 将内容反馈给客户端
		fmt.Fprintf(w, "你好 %s，你输入的内容是：%s", userName, userText)
	} else {
		fmt.Fprintf(w, redirectHTML)
		}
}

func main() {
	http.HandleFunc("/", index)              // 设置访问的路由
	http.HandleFunc("/page", page)           // 设置访问的路由
	err := http.ListenAndServe(":9090", nil) // 设置监听的端口
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}





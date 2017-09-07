package main
import (
	"html/template"
	"log"
	"net/http"
)
//定义输出错误
func checkErr(err error) {
	if err != nil {
		log.Println(err)
	}
}

//定义数据类型
type UserData struct {
	Name string
	Text string
}

func renderHTML(w http.ResponseWriter, file string, data interface{}) {
	// 获取页面内容
	t, err := template.New(file).ParseFiles("views/" + file)
	checkErr(err)
	// 将页面渲染后反馈给客户端
	t.Execute(w, data)
}


func page(w http.ResponseWriter, r *http.Request){
	if r.Method == "POST" {
	if err := r.ParseForm(); err != nil {
			log.Println("Handler:page:ParseForm: ", err)
		}
	u := UserData{}
	u.Name = r.Form.Get("username")
	u.Text = r.Form.Get("usertext")

	renderHTML(w, "page.html", u)
	} else {
		renderHTML(w, "redirect.html", "/")
	}
}


func index(w http.ResponseWriter, r *http.Request) {
	// 渲染页面并输出
	renderHTML(w, "index.html", "no data")
}

func main() {
	http.HandleFunc("/", index)              // 设置访问的路由
	http.HandleFunc("/page", page)           // 设置访问的路由
	err := http.ListenAndServe(":9090", nil) // 设置监听的端口
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

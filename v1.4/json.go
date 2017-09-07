package main
//json,net
import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

//定义item结构体
type Item struct {
	Seq    int
	Result map[string]int
}

//定义message结构体
type Message struct {
	Dept    string
	Subject string
	Time    int64
	Detail  []Item
}

func getJson() ([]byte, error) {
	//使用集合
	pass := make(map[string]int)

	pass["x"] = 50
	pass["c"] = 60
	item1 := Item{100, pass}

	reject := make(map[string]int)
	reject["l"] = 11
	reject["d"] = 20
	item2 := Item{200, reject}
	
	detail := []Item{item1, item2}
	//构建数组

	m := Message{"IT", "KPI", time.Now().Unix(), detail}
	return json.MarshalIndent(m, "", "")
}

//调用组装好json逻辑的函数
func handler(w http.ResponseWriter, r *http.Request) {
	resp, err := getJson()
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(w, string(resp))
}

func main() {
	http.HandleFunc("/", handler)
	//定义路由并且调用封装好的json函数
	http.ListenAndServe(":9090", nil)
}

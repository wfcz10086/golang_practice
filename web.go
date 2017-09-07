
package main

import (
	"fmt"
	"log"
	"net/http"
)

func index(w http.ResponseWriter,r *http.Request){
	fmt.Fprintf(w, "Hello World!")
}

func main(){
	http.HandleFunc("/", index) 
	err := http.ListenAndServe(":9090", nil)
	if err !=nil {
		log.Fatal("ListenAndServe: ", err)
	}
}


package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
)

var (
	port    = flag.String("port", ":8080", "default listen port")
	client1 *http.Client
	client2 *http.Client
)

type Reply map[string]interface{}
type MSGResult struct {
	MSG string
	Err error
}

func WriteJSON(w http.ResponseWriter, code int, v interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(v)
}

func doGet(client *http.Client, u string) ([]byte, error) {
	resp, err := client.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func Get(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	result := make(chan MSGResult, 2)
	wg := sync.WaitGroup{}
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			var client *http.Client
			u := "http://127.0.0.1:8080/msg?digi="
			if idx == 0 {
				client = client1
				u += "100"
			} else {
				client = client2
				u += "200"
			}
			reply, err := doGet(client, u)
			log.Infof("get reply for idx=%d, reply=%s, err=%v", idx, string(reply), err)
			if err != nil {
				result <- MSGResult{MSG: "", Err: err}
			} else {
				result <- MSGResult{MSG: string(reply), Err: nil}
			}
		}(i)
	}
	go func() {
		wg.Wait()
		close(result)
	}()

	ret := []string{}
	for res := range result {
		ret = append(ret, res.MSG)
	}

	WriteJSON(w, http.StatusOK, Reply{"msg": ret})
	return
}

func Post(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	WriteJSON(w, http.StatusOK, Reply{"msg": "this is post"})
	return
}

func Msg(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	digi := r.FormValue("digi")
	WriteJSON(w, http.StatusOK, Reply{"msg": fmt.Sprintf("digi=%s", digi)})
	return
}

func main() {
	flag.Parse()
	router := httprouter.New()
	router.GET("/get", Get)
	router.GET("/msg", Msg)
	router.POST("/post", Post)

	client1 = &http.Client{
		Timeout: 15 * time.Millisecond,
	}

	client2 = &http.Client{
		Timeout: 10 * time.Millisecond,
	}

	log.Fatal(http.ListenAndServe(*port, router))
}

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

type Resp struct {
	Code int               `json:"code"`
	Msg  string            `json:"msg"`
	Data map[string]string `json:"data"`
}

var StatusErr = errors.New("status code not 200")

func request(method string, path string, body []byte) (*Resp, error) {
	req, err := http.NewRequest(method, path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		_, _ = io.Copy(ioutil.Discard, resp.Body)
		return nil, StatusErr
	}

	bts, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var ret Resp
	if err := json.Unmarshal(bts, &ret); err != nil {
		return nil, err
	}

	return &ret, nil
}

const (
	host = "http://localhost:8080"
)

func set(key, value string, expire ...int64) error {
	req := map[string]interface{}{}
	req[key] = value
	if len(expire) > 0 {
		req["expire"] = expire[0]
	}
	body, _ := json.Marshal(&req)
	resp, err := request("POST", host+"/api/cache/"+key, body)
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return errors.New(resp.Msg)
	}

	return nil

}

func get(key string) (string, error) {
	resp, err := request("GET", host+"/api/cache/"+key, nil)
	if err != nil {
		return "", err
	}

	if resp.Code != 0 {
		return "", errors.New(resp.Msg)
	}

	return resp.Data[key], nil

}

func del(key string) error {

	resp, err := request("DELETE", host+"/api/cache/"+key, nil)
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return errors.New(resp.Msg)
	}

	return nil
}

func main() {

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go testKey(fmt.Sprintf("key%d", i), &wg)
	}

	wg.Wait()

	log.Println("恭喜，完成作业")
}

func testKey(key string, wg *sync.WaitGroup) {
	defer wg.Done()
	if _, err := get(key); err == nil {
		log.Fatalf("未设置%s的值，应返回错误", key)
	}

	if err := set(key, "val1"); err != nil {
		log.Fatalf("设置%s失败:%v", key, err)
	}

	if val, err := get(key); err != nil || val != "val1" {
		log.Fatalf("获取%s失败，或值不等于设置的值", key)
	}

	if err := set(key, "val2", time.Now().Add(time.Second).Unix()); err != nil {
		log.Fatalf("设置%s失败:%v", key, err)
	}

	if val, err := get(key); err != nil || val != "val2" {
		log.Fatalf("获取%s失败，或值不等于设置的值", key)
	}

	time.Sleep(time.Second)

	if _, err := get(key); err == nil {
		log.Fatalf("%s的过期，应返回错误", key)
	}

}

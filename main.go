package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func main() {
	// 	请开始你的表演
	fmt.Println("服务启动.....")
	c := new(Cache)
	c.store = make(map[string]string)
	http.HandleFunc("/api/cache/", c.cacheHandler)
	_ = http.ListenAndServe(":8080", nil)
}

type Resp struct {
	Code int               `json:"code"`
	Msg  string            `json:"msg"`
	Data map[string]string `json:"data"`
}

type Cache struct {
	store map[string]string
}

func (c *Cache) cacheHandler(w http.ResponseWriter, req *http.Request) {
	method := req.Method   //获取请求方法
	url := req.RequestURI	//获取请求的url
	w.Header().Set("Content-Type", "application/json") //设置响应头
	key, expire := GetKeyFromUrl(url)	//提取key,和expire

	if key == "" && expire == "" {
		//访问的url不合法，拒绝访问
		w.WriteHeader(403)
	}

	switch method {
		case "POST":
			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				fmt.Printf("获取请求体失败, %v\n", err)
				return
			}
			strBody := string(body)
			if expire == "" { //这个参数为空，表示设置新的缓存
				code, msg, data := c.SetCache(key, strBody)
				w.Write(StuctToJson(code, msg, data))
			} else { //设置过期时间
				if expire == "expire" {
					code, msg, data := c.SetCacheExpire(key, strBody)
					w.Write(StuctToJson(code, msg, data))
				} else {
					//第4个参数不正确，拒绝访问
					w.WriteHeader(403)
				}
			}
		case "GET":
			code, msg, data, status := c.GetCacheByKey(key)
			if status != 200 {
				w.WriteHeader(status)
			} else {
				w.Write(StuctToJson(code, msg, data))
			}
		case "DELETE":
			code, msg, data := c.DelCacheByKey(key)
			w.Write(StuctToJson(code, msg, data))
		default:
			//请求方式不正确，返回405
			w.WriteHeader(405)
	}
}

//设置缓存
func (c *Cache) SetCache(key string, body string) (int, string, map[string]string) {
	c.store[key] = body
	if c.store[key] == "" {
		return 1, "设置缓存失败", nil
	}
	return 0, "设置缓存成功", nil
}

//获取缓存
func (c *Cache) GetCacheByKey(key string) (int, string, map[string]string, int) {
	if c.store[key] == "" { //找不到缓存
		return 0, "", nil, 404
	}

	value := c.store[key]
	mapValue := JsonToMap(value)
	now := time.Now().Unix()

	intTime, _ := strconv.ParseInt(mapValue["expire"], 10, 64)
	if now > intTime { //过期的缓存
		return 0, "", nil, 404
	}

	data := make(map[string]string)
	data[key] = mapValue["key"]
	return 0, "获取缓存成功", data, 200
}

//删除缓存
func (c *Cache) DelCacheByKey(key string) (int, string, map[string]string) {
	if c.store[key] == "" {
		return 1, "删除缓存错误，缓存不存在", nil
	}

	delete(c.store, key)
	if c.store[key] == "" {
		return 0, "删除缓存成功", nil
	}

	return 1, "删除缓存失败", nil
}

//给缓存设置过期时间
func (c *Cache) SetCacheExpire(key string, expire string) (int, string, map[string]string) {
	if c.store[key] == "" {
		return 1, "设置缓存过期时间错误，缓存不存在", nil
	}

	expireMap := JsonToMap(expire)
	value := c.store[key]
	mapValue := JsonToMap(value)
	mapValue["expire"] = expireMap["expire"]
	res, err := json.Marshal(mapValue)
	if err != nil {
		fmt.Printf("map转json失败: %+v\n", err)
		return 1, "设置缓存过期时间失败", nil
	}
	c.store[key] = string(res)
	return 0, "设置缓存过期时间成功", nil
}

//json字符串转map
func JsonToMap(jsonStr string) map[string]string {
	m := make(map[string]string)
	err := json.Unmarshal([]byte(jsonStr), &m)
	if err != nil {
		fmt.Printf("json转map失败: %+v\n", err)
		return nil
	}
	return m
}

//转json字符串
func StuctToJson(code int, msg string, data map[string]string) []byte {
	rsp := new(Resp)
	rsp.Code = code
	rsp.Msg = msg
	rsp.Data = data
	res, err := json.Marshal(rsp)
	if err != nil {
		fmt.Printf("转json字符串失败: %+v\n", err)
		return nil
	}
	return res
}

//从url中解析出key和expire
func GetKeyFromUrl(url string) (string, string) {
	strArr := strings.Split(strings.Trim(url, "/"), "/")
	if !(len(strArr) == 3 || len(strArr) == 4) {
		return "", ""
	}
	if len(strArr) == 3 {
		key := strArr[2]
		return key, ""
	}
	return strArr[2], strArr[3]
}

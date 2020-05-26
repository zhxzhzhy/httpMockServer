package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"
)

var tag int64

func main() {
	port := os.Args[1]
	http.HandleFunc("/", mockServer)
	err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	if err != nil {
		logPrint("main... ListenAndServe: ", err)
	}
}

func mockServer(w http.ResponseWriter, req *http.Request) {
	tag = time.Now().UnixNano()
	reqPort := strings.Split(req.Host, ":")[1]
	mockConf, err := getMockConf(reqPort)
	if err != nil {
		logPrint(fmt.Sprintf("mockServer... " + err.Error()))
		_, _ = w.Write([]byte("获取mock配置失败: " + err.Error()))
		return
	}
	respBytes, err := doMatchReqAndConf(req, mockConf)
	if err != nil {
		logPrint(w.Write([]byte("匹配失败: " + err.Error())))
		return
	}
	logPrint(w.Write(respBytes))
}

func getMockConf(port string) (mockConf *MockConf, err error) {
	c, err := redis.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		logPrint("getMockConf... Connect to redis error", err)
		return nil, err
	}
	defer c.Close()
	logPrint(fmt.Sprintf("getMockConf... 执行redis命令: get mock_server:%s", port))
	confBytes, err := redis.Bytes(c.Do("get", fmt.Sprintf("mock_server:%s", port)))
	if err != nil {
		logPrint(fmt.Sprintf("getMockConf... 执行失败,错误为: %s", err.Error()))
		return nil, err
	}
	if confBytes == nil {
		logPrint("getMockConf... 未获取到数据")
		return nil, errors.New("未读取到数据")
	}
	logPrint("getMockConf... " + string(confBytes))
	err = json.Unmarshal(confBytes, &mockConf)
	return
}

func doMatchReqAndConf(req *http.Request, mockConf *MockConf) ([]byte, error) {
	// 从上到下逐个匹配
	url := strings.Trim(req.RequestURI, "/")
	method := req.Method
	var bodyBytes []byte
	if strings.ToUpper(method) == "POST" {
		if bytes, err := ioutil.ReadAll(req.Body); err != nil {
			logPrint("doMatchReqAndConf... 读取post请求消息体失败,错误为: " + err.Error())
			return nil, err
		} else {
			bodyBytes = bytes
		}
	}
	logPrint(fmt.Sprintf(" url: %s, method: %s, body: %s", url, method, bodyBytes))
	for _, mock := range mockConf.Mocks {
		logPrint("尝试匹配: ", mock)
		//1. 匹配url
		if len(mock.Url) > 0 && mock.Url != "*" {
			// 需要做匹配
			if strings.ToUpper(mock.Url) != strings.ToUpper(url) {

				logPrint(fmt.Sprintf("url(%s, %s)不匹配，过滤掉...  ", mock.Url, url))
				continue
			}
		}
		// 2. 匹配请求方法
		if len(mock.Method) > 0 && strings.ToUpper(mock.Method) != strings.ToUpper(method) {
			logPrint("请求方法不匹配，过滤掉... ")
			continue
		}
		// 3. 匹配请求参数
		if strings.ToUpper(method) == "POST" && !doMatchParam(mock.ReqBody, bodyBytes) {
			logPrint("参数不匹配，过滤掉... ")
			continue
		}
		// 找到了匹配的mock
		logPrint(fmt.Sprintf("找到了匹配的mock: %v", mock.Resp))
		return json.Marshal(mock.Resp)
	}
	return nil, errors.New("未找到合适的记录")
}

func doMatchParam(mockBody map[string]interface{}, reqBodyBytes []byte) bool {
	var reqBody interface{}
	if err := json.Unmarshal(reqBodyBytes, &reqBody); err != nil {
		logPrint("反序列化请求消息体失败: " + string(reqBodyBytes) + " " + err.Error())
		return false
	}
	for fullPath, refValue := range mockBody {
		paths := strings.Split(fullPath, "->")
		reqValue := findValueByPath(reqBody, paths)
		if reqValue == nil {
			return false
		}
		if reflect.TypeOf(refValue) != reflect.TypeOf(reqValue) {
			return false
		}
		if fmt.Sprintf("%v", reqValue) != fmt.Sprintf("%v", refValue) {
			return false
		}
	}
	return true
}

func findValueByPath(body interface{}, paths []string) interface{} {
	if body == nil {
		return nil
	}
	if len(paths) == 0 {
		return nil
	}
	if len(paths) == 1 {
		return body.(map[string]interface{})[paths[0]]
	}
	tmp := body.(map[string]interface{})[paths[0]]
	return findValueByPath(tmp, paths[1:])
}

func logPrint(a ...interface{}) {
	now := time.Now().Format("2006-01-02 15:04:05")
	fmt.Print("[", tag, "][", now, "] ")
	fmt.Println(a...)
}

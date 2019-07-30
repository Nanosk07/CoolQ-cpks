package testapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func RequestAPI(method, url, body string) string {
	/* 确定请求方法 */
	request_method := http.MethodGet
	if strings.ToUpper(method) == "POST" {
		request_method = http.MethodPost
	} else if strings.ToUpper(method) == "DELETE" {
		request_method = http.MethodDelete
	} else if strings.ToUpper(method) == "PATCH" {
		request_method = http.MethodPatch
	} else if strings.ToUpper(method) == "PUT" {
		request_method = http.MethodPut
	}

	/* 确定请求地址 */
	request_url := url
	if !strings.HasPrefix(url, "http") {
		request_url = "http://" + url
	}

	/* 确定请求内容 */
	request_body := body
	if body != "" {
		/*
			updateParams := `{"testKey":"` + query + `"}` // 使用转义反引号完成json转换
			req, _ := http.NewRequest(request_method, CONFIG.ServerUrl, bytes.NewBuffer([]byte(updateParams)))
			req.Header.Set("Content-Type", "application/json")
		*/
		body += body
	}
	fmt.Println(request_body)

	/* 构造请求 */
	req, _ := http.NewRequest(request_method, request_url, nil)

	/* 发送请求并获取响应 */
	client := &http.Client{}
	resp, _ := client.Do(req)

	/* 解析响应 */
	fmt.Println("response Status:", resp.Status)
	//fmt.Println("response Headers:", resp.Header)
	resp_body, _ := ioutil.ReadAll(resp.Body)
	return string(resp_body)
}

func JsonToMap(json_string string) map[string]interface{} {
	/* json(string) 转为 map */
	var result map[string]interface{}
	err := json.Unmarshal([]byte(json_string), &result);
	if err != nil {
		errorMap := make(map[string]interface{})
		errorMap["error"] = err
		return errorMap
	}
	return result
}
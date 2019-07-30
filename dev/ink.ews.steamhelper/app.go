package main

import (
	"encoding/json"
	"fmt"
	"github.com/Tnze/CoolQ-Golang-SDK/cqp"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

//go:generate cqcfg .
// cqp: 名称: Steam助手-Alpha
// cqp: 版本: 0.7.1:1
// cqp: 作者: Rakuyo(imwhtl@gmail.com)
// cqp: 简介: 提供steam相关查询以及将ASF接入酷Q
func main() {}
func init() {
	cqp.AppID = "ink.ews.steamhelper"
	cqp.Enable = onEnable
	cqp.Disable = onDisable
	cqp.GroupMsg = onGroupMsg
	cqp.PrivateMsg = onPrivateMsg
}

/**
 * 绑定cq事件函数
 * func onEnable() int32     => 启用插件
 * func onGroupMsg() int32   => 群消息事件
 * func onPrivateMsg() int32 => 私聊事件
 * func onDisable() int32    => 停用插件 //TODO 未生效
 */

func onEnable() int32 {
	defer handleErr()
	cqp.AddLog(cqp.Debug, "启用插件", "插件启用: Steam助手(ink.ews.steamhelper)")
	applyConfig()
	//startPolling(5)
	return 0
}

func onGroupMsg(subType, msgID int32, fromGroup, fromQQ int64, fromAnonymous, msg string, font int32) int32 {
	defer handleErr()
	reply := replyMsg(msg, fromQQ)
	if reply != "" {
		cqp.SendGroupMsg(fromGroup, "[CQ:at,qq="+strconv.FormatInt(fromQQ, 10)+"] "+reply)
		return 1
	}
	return 0
}

func onPrivateMsg(subType, msgID int32, fromQQ int64, msg string, font int32) int32 {
	defer handleErr()
	reply := replyMsg(msg, fromQQ)
	if reply != "" {
		cqp.SendPrivateMsg(fromQQ, reply)
		return 1
	}
	return 0
}

func onDisable() int32 {
	defer handleErr()
	saveConfig()
	cqp.AddLog(cqp.Debug, "停用插件", "插件停用: Steam助手(ink.ews.steamhelper)") /* BUG: 并不会显示 */
	return 0
}

/**
 * 主要业务逻辑
 * func startPolling() => 并发调用匿名轮询函数
 * func replyMsg()      => 处理消息
 */

func replyMsg(msg string, fromQQ int64) string {
	var reply string
	if fromQQ != CONFIG.MasterQQ {
		return ""
	}
	if strings.HasPrefix(msg, "/steam") {
		command := strings.Trim(strings.TrimPrefix(msg, "/steam"), " ")
		reply = querySteam(command)
	} else if strings.HasPrefix(msg, "/asf") {
		command := strings.Trim(strings.TrimPrefix(msg, "/asf"), " ")
		reply = queryASF(command)
	} else if strings.HasPrefix(msg, "/bot ") {
		//command := strings.Trim(strings.TrimPrefix(msg, "/bot"), " ")
		//reply = queryBot(command)
	} else if strings.HasPrefix(msg, "!") {
		command := strings.TrimPrefix(msg, "!")
		reply = query(command)
	}
	return reply
}

func querySteam(query_str string) string {
	if query_str != "" {
		query_str = "，因此" + query_str + "指令无效"
	}
	return "暂时未做/steam的功能" + query_str
	//url := "api.steam.com"
	//resp_body, resp_status := requestAPI("GET", generateURL(url)+query_str, "")
	//cqp.AddLog(cqp.Debug, "querySteam()", string(resp_body))
	//return processJson(resp_body, resp_status)
}

func queryASF(query_str string) string {
	query_method := "POST"
	if strings.ToLower(query_str) == "exit" {
		query_str = "Exit"
	} else if strings.ToLower(query_str) == "restart" {
		query_str = "Restart"
	} else if strings.ToLower(query_str) == "update" {
		query_str = "Update"
	} else {
		query_method = "GET"
		query_str = ""
	}
	resp_body, resp_status := requestAPI(query_method, generateASFURL("asf")+query_str, "")
	cqp.AddLog(cqp.Debug, "queryASF()", string(resp_body))
	return processJson(resp_body, resp_status)
}

func queryBot(query_str string) string {
	var RequestBody string
	args := strings.SplitN(query_str, " ", 2)
	query_str = args[0]
	query_method := "POST"
	if len(args) > 1 {
		command := strings.ToLower(args[1])
		command_list := []string{"pause", "redeem", "rename", "resume", "start", "stop"}
		if strings.ToLower(command) == "bgr" {
			query_method = "GET"
			query_str += "/GamesToRedeemInBackground"
		} else if in_array(command, command_list) {
			query_str += "/" + command
			//TODO 需要额外带参 详见/swagger/index.html
			RequestBody = "args"
		}
	} else {
		query_method = "GET"
	}
	resp_body, resp_status := requestAPI(query_method, generateASFURL("bot")+query_str, RequestBody)
	cqp.AddLog(cqp.Debug, "queryBot()", string(resp_body))
	return processJson(resp_body, resp_status)
}

func query(query_str string) string {
	resp_body, resp_status := requestAPI("POST", generateASFURL("")+query_str, "")
	cqp.AddLog(cqp.Debug, "query()", string(resp_body))
	resp_msg := resp_status
	var result CommandResponseMsg
	if err := json.Unmarshal([]byte(resp_body), &result); err == nil {
		if resp_status == "200 OK" {
			resp_msg = result.Result
		}
	}
	return resp_msg
}

func processJson(resp_body []byte, resp_status string) string {
	resp_msg := resp_status
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(resp_body), &result); err == nil && resp_status == "200 OK" {
		if _, ok := result["Result"]; ok {
			if resp_result_json, err := json.MarshalIndent(result["Result"], "", "\t"); err == nil {
				resp_msg = string(resp_result_json)
			} else {
				cqp.AddLog(cqp.Warning, "打印错误", fmt.Sprint(err))
			}
		} else if result["Success"].(bool) == true {
			resp_msg = result["Message"].(string)
		}
	}
	return resp_msg
}

/**
 * 次要业务辅助
 * func generateURL() string => 根据用户配置生成数据接口URL
 */

func in_array(search string, array []string) bool {
	for _, v := range (array) {
		if v == search {
			return true
		}
	}
	return false
}

func generateASFURL(keyword string) string {
	url := CONFIG.IPCUrl
	url = generateURL(url) + "Api/"
	kEYWORD := strings.ToUpper(keyword)
	if kEYWORD == "ASF" {
		url += "ASF/"
	} else if kEYWORD == "BOT" {
		url += "Bot/"
	} else {
		url += "Command/"
	}
	return url
}

func generateURL(url string) string {
	if !strings.HasPrefix(url, "http") {
		url = "http://" + url
	}
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	return url
}

func requestAPI(method, url, json string) ([]byte, string) {
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
	req, _ := http.NewRequest(request_method, url, nil)
	if CONFIG.IPCPassword != "" {
		req.Header.Set("Authentication", CONFIG.IPCPassword)
	}
	client := &http.Client{}
	resp, _ := client.Do(req)
	body, _ := ioutil.ReadAll(resp.Body)
	return body, resp.Status
}

/**
 * 全局变量&数据类型的声明/定义
 * var CONFIG ConfigStruct  => 全局变量: 用户配置
 * type ConfigStruct struct => struct: 用户配置
 */

var CONFIG ConfigStruct

type ConfigStruct struct {
	MasterQQ    int64  `json:"master_qq"`
	QQGroup     int64  `json:"qq_group_num"`
	IPCUrl      string `json:"ipc_url"`
	IPCPassword string `json:"ipc_password"`
}

type CommandResponseMsg struct {
	Result  string
	Message string
	Success bool
}

/**
 * 基础工具函数
 * func applyConfig()                => 应用用户配置
 * func readConfig() ([]byte, error) => 读取用户配置文件
 * func saveConfig()                 => 保存用户配置文件
 * func getFile() (string, error)    => 获取插件相关文件，若不存在则顺便创建
 * func printErr()                   => 打印错误到日志
 * func handleErr()                  => 打印致命错误到日志
 */

func applyConfig() {
	configdata, err := readConfig("config.json")
	if err != nil {
		printErr(err)
	} else {
		err = json.Unmarshal(configdata, &CONFIG)
	}
	//cqp.AddLog(cqp.Debug, "打印数据", "读取到配置: "+fmt.Sprint(CONFIG))
}

func readConfig(config_file string) ([]byte, error) {
	var configdata []byte
	file, err := getFile(config_file)
	if err != nil {
		return configdata, err
	}
	configdata, err = ioutil.ReadFile(file)
	if os.IsNotExist(err) {
		return configdata, nil
	} else if err != nil {
		return configdata, err
	}
	cqp.AddLog(cqp.Debug, "读取配置", "读取配置文件: "+file)
	return configdata, nil
}

func saveConfig() {
	configsdata, err := json.MarshalIndent(CONFIG, "", "\t")
	if err != nil {
		printErr(err)
	}
	file, err := getFile("config.json")
	if err != nil {
		printErr(err)
	}
	//cqp.AddLog(cqp.Debug, "打印数据", "保存的配置: "+fmt.Sprint(CONFIG))
	//cqp.AddLog(cqp.Debug, "打印数据", "保存的配置(json): "+string(configsdata))
	cqp.AddLog(cqp.Debug, "保存配置", "保存配置文件: "+file)
	err = ioutil.WriteFile(file, configsdata, 0666)
	if err != nil {
		printErr(err)
	}
}

func getFile(name string) (string, error) {
	appdir := cqp.GetAppDir()
	file_path := filepath.Join(appdir, name)
	if err := os.MkdirAll(appdir, os.ModeDir); err != nil {
		return file_path, err
	}
	_, err := os.Stat(file_path)
	if os.IsNotExist(err) {
		file, err := os.Create(file_path)
		if err != nil {
			printErr(err)
		}
		defer file.Close()
	}
	return file_path, nil
}

func printErr(err error) {
	cqp.AddLog(cqp.Error, "错误", err.Error())
}

func handleErr() {
	if err := recover(); err != nil {
		cqp.AddLog(cqp.Fatal, "严重错误", fmt.Sprint(err))
	}
}

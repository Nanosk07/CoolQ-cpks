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
// cqp: 名称: 儒雅随和-Alpha
// cqp: 版本: 0.7.1:1
// cqp: 作者: Rakuyo(imwhtl@gmail.com)
// cqp: 简介: 慎用，嘴太臭了一举报一个准，机器人被封后果自负。只对[么么哒/cnm/nmsl/申请对线]有反应。进入对线状态会追着喷，直到一方道歉结束对线。
func main() {}
func init() {
	cqp.AppID = "ink.ews.ruyasuihe"
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
	cqp.AddLog(cqp.Debug, "启用插件", "插件启用: 儒雅随和(ink.ews.ruyasuihe)")
	AT_ME = atQQ(cqp.GetLoginQQ())
	applyConfig()
	return 0
}

func onGroupMsg(subType, msgID int32, fromGroup, fromQQ int64, fromAnonymous, msg string, font int32) int32 {
	defer handleErr()
	check_enemy := outputDamage(fromQQ, msg)
	if check_enemy == 1 {
		cqp.SendGroupMsg(fromGroup, atQQ(CONFIG.EnemyQQ)+generateBee(CONFIG.FireLevel))
		Wire2Wire += 1
		return 1
	} else if check_enemy == -1 {
		cqp.SendGroupMsg(fromGroup, atQQ(fromQQ)+"嘻嘻，这次就算了。")
		Wire2Wire = 0
		CONFIG.FireLevel = 0
		saveConfig()
		return 1
	}
	if !strings.Contains(msg, AT_ME) {
		return 0
	}
	reply := replyMsg(msg, fromQQ)
	if reply != "" {
		cqp.SendGroupMsg(fromGroup, atQQ(fromQQ)+reply)
		//cqp.SendGroupMsg(fromGroup, AT_ME+reply)
		return 1
	}
	return 0
}

func onPrivateMsg(subType, msgID int32, fromQQ int64, msg string, font int32) int32 {
	defer handleErr()
	check_enemy := outputDamage(fromQQ, msg)
	if check_enemy == 1 {
		cqp.SendPrivateMsg(fromQQ, generateBee(CONFIG.FireLevel))
		Wire2Wire += 1
		return 1
	} else if check_enemy == -1 {
		cqp.SendPrivateMsg(fromQQ, "嘻嘻，这次就算了。")
		Wire2Wire = 0
		CONFIG.FireLevel = 0
		saveConfig()
		return 1
	}
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
	cqp.AddLog(cqp.Debug, "停用插件", "插件停用: 儒雅随和(ink.ews.ruyasuihe)") /* BUG: 并不会显示 */
	return 0
}

/**
 * 主要业务逻辑
 * func startPolling() => 并发调用匿名轮询函数
 * func replyMsg()      => 处理消息
 */

func outputDamage(fromQQ int64, msg string) int {
	if Wire2Wire > 0 && CONFIG.FireLevel == 0 {
		CONFIG.FireLevel = 1
		saveConfig()
	}
	if fromQQ == CONFIG.EnemyQQ {
		if msg == "对不起" || msg == "我错了" || msg == "dbq" {
			CONFIG.EnemyQQ = 0
			saveConfig()
			return -1
		} else {
			cqp.AddLog(cqp.Debug, "对线情况", "Wire2Wire="+fmt.Sprint(Wire2Wire)+"    CONFIG.FireLevel="+fmt.Sprint(CONFIG.FireLevel))
			return 1
		}
	}
	return 0
}

func replyMsg(msg string, fromQQ int64) string {
	var reply string
	msg = strings.Trim(msg, " ")
	msg = strings.Trim(msg, AT_ME)
	msg = strings.Trim(msg, " ")
	if strings.Contains(msg, "申请对线") {
		if CONFIG.EnemyQQ == 0 {
			CONFIG.EnemyQQ = fromQQ
			reply = generateBee(CONFIG.FireLevel)
		} else {
			reply = "别急着送[CQ:emoji,id=128052]，一个一个来。"
		}
	} else if strings.Contains(msg, "cnm") {
		reply = generateBee(CONFIG.FireLevel)
	} else if strings.Contains(msg, "nmsl") {
		reply = generateBee(1)
	} else if strings.Contains(msg, "么么哒") {
		reply = generateBee(-1)
	}
	return reply
}

/**
 * 次要业务辅助
 * func generateURL() string => 根据用户配置生成请求数据接口的URL
 */

func generateBee(level int) string {
	url := "https://nmsl.shadiao.app/api.php?level=min"
	if level == 1 {
		url = "https://nmsl.shadiao.app/api.php"
	} else if level == -1 {
		url = "https://chp.shadiao.app/api.php"
	}
	req, _ := http.NewRequest("GET", url, nil)
	client := &http.Client{}
	resp, _ := client.Do(req)
	body, _ := ioutil.ReadAll(resp.Body)
	return string(body)
}

func atQQ(qq int64) string {
	return "[CQ:at,qq=" + strconv.FormatInt(qq, 10) + "] "
}

/**
 * 全局变量&数据类型的声明/定义
 * var CONFIG ConfigStruct  => 全局变量: 用户配置
 * type ConfigStruct struct => struct: 用户配置
 */

var AT_ME string
var Wire2Wire int = 0
var CONFIG ConfigStruct

type ConfigStruct struct {
	MasterQQ  int64 `json:"master_qq"`
	QQGroup   int64 `json:"qq_group_num"`
	EnemyQQ   int64 `json:"enemy_qq"`
	FireLevel int   `json:"fire_level"`
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

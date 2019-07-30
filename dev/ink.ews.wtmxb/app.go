package main

import (
	"encoding/json"
	"fmt"
	"github.com/Tnze/CoolQ-Golang-SDK/cqp"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

//go:generate cqcfg .
// cqp: 名称: 下班预热-Alpha
// cqp: 版本: 0.7.1:1
// cqp: 作者: Rakuyo(imwhtl@gmail.com)
// cqp: 简介: 沙雕群友定制。[/help 下班][预热]
func main() {}
func init() {
	cqp.AppID = "ink.ews.wtmxb"
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
	cqp.AddLog(cqp.Debug, "启用插件", "插件启用: 下班预热(ink.ews.wtmxb)")
	AT_ME = atQQ(cqp.GetLoginQQ())
	applyConfig()
	if err := readTimeSheets(); err != nil {
		printErr(err)
		return -1
	}
	return 0
}

func onGroupMsg(subType, msgID int32, fromGroup, fromQQ int64, fromAnonymous, msg string, font int32) int32 {
	defer handleErr()
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
	saveTimeSheets()
	cqp.AddLog(cqp.Debug, "停用插件", "插件停用: 下班预热(ink.ews.wtmxb)") /* BUG: 并不会显示 */
	return 0
}

/**
 * 主要业务逻辑
 * func startPolling() => 并发调用匿名轮询函数
 * func replyMsg()      => 处理消息
 */

func replyMsg(msg string, fromQQ int64) string {
	var reply string
	queryer, ok := timeSheets[int64(fromQQ)]
	if strings.HasPrefix(msg, "/help 下班") {
		if ok && queryer.ShiftEnds != "" {
			reply = "你设置的下班时间为" + queryer.ShiftEnds[0:2] + "时" + queryer.ShiftEnds[3:5] + "分。"
		} else {
			reply = "你还没设置过下班时间。"
		}
		reply += "\n/set 下班 [HH:mm] 设置下班时间"
		return reply
	} else if strings.HasPrefix(msg, "/set 下班 ") {
		quit_setting := strings.TrimPrefix(msg, "/set 下班 ")
		quit_setting = strings.Replace(quit_setting, "：", ":", -1)
		cqp.AddLog(cqp.Debug, "输入格式", fmt.Sprint(len(quit_setting)))
		if len(quit_setting) != 5 {
			return "请输入正确的时间格式。"
		}
		quit_hour, hour_err := strconv.Atoi(quit_setting[0:2])
		quit_minute, minute_err := strconv.Atoi(quit_setting[3:5])
		if hour_err != nil || minute_err != nil {
			reply = "输入数字，你TM懂不懂什么叫数字。\n/set 下班 18:00\n这个格式"
		} else if quit_hour < 0 || quit_hour > 23 {
			reply = "你一天80个小时。"
		} else if
		quit_minute < 0 || quit_minute > 59 {
			reply = "你一个小时3000分钟。"
		} else {
			queryer.ShiftEnds = quit_setting[0:2] + ":" + quit_setting[3:5]
			timeSheets[int64(fromQQ)] = queryer
			saveTimeSheets()
			reply = "下班时间已设置为每天的" + quit_setting[0:2] + "时" + quit_setting[3:5] + "分。"
		}
		return reply
	}
	Weekday := time.Now().Weekday().String()
	if Weekday == "Saturday" {
		if msg == "预热" {
			reply = "宁也是996¿"
		}
	} else if Weekday == "Sunday" {
		if msg == "预热" {
			reply = "你预你[CQ:emoji,id=128052]呢，今天星期天。"
		}
	} else {
		if msg == "预热" {
			if ok && queryer.ShiftEnds != "" {
				reply = calculateTime(queryer)
			} else {
				reply = "你预几把，你还没设置过下班时间。\n/set 下班 [HH:mm] 设置下班时间"
			}
		} else if strings.Contains(msg, "准备下班") || strings.Contains(msg, "准备跑") || strings.Contains(msg, "准备溜") {
			if ok && queryer.ShiftEnds != "" {
				reply = calculateTime(queryer)
			} else {
				reply = "别准备了，你还没设置过下班时间。\n/set 下班 [HH:mm] 设置下班时间"
			}
		} else if msg == "下班" {
			if ok && queryer.ShiftEnds != "" {
				reply = calculateTime(queryer)
			} else {
				reply = "下锤子，你还没设置过下班时间。\n/set 下班 [HH:mm] 设置下班时间"
			}
		}
	}
	return reply
}

func calculateTime(queryer TimeSheetStruct) string {
	Now := time.Now()
	cqp.AddLog(cqp.Debug, "打印数据", fmt.Sprint(queryer))
	loc, _ := time.LoadLocation("Local")
	today_quit_string := Now.Format("2006-01-02 ") + queryer.ShiftEnds[0:2] + ":" + queryer.ShiftEnds[3:5]
	today_quit_time, _ := time.ParseInLocation("2006-01-02 15:04", today_quit_string, loc)
	var calculateResult string
	if Now.Unix() > today_quit_time.Unix() {
		work_over := math.Floor(Now.Sub(today_quit_time).Minutes() + 0.5)
		work_over_int, _ := strconv.Atoi(strconv.FormatFloat(work_over, 'f', -1, 64))
		if (work_over_int == 0) {
			return "该下班了。"
		}
		calculateResult = "醒醒，你已经加班"
		if work_over_int > 60 {
			calculateResult += strconv.Itoa(work_over_int/60) + "小时"
		}
		calculateResult += strconv.Itoa(work_over_int%60) + "分钟了。"
	} else {
		remain_minute := math.Floor(today_quit_time.Sub(Now).Minutes() + 0.5)
		remain_int, _ := strconv.Atoi(strconv.FormatFloat(remain_minute, 'f', -1, 64))
		if (remain_minute == 0) {
			return "该下班了。"
		}
		calculateResult = "你还剩"
		if remain_int > 60 {
			calculateResult += strconv.Itoa(remain_int/60) + "小时"
		}
		calculateResult += strconv.Itoa(remain_int%60) + "分钟下班。"
	}
	return calculateResult
}

/**
 * 次要业务辅助
 * func generateURL() string => 根据用户配置生成请求数据接口的URL
 */

func atQQ(qq int64) string {
	return "[CQ:at,qq=" + strconv.FormatInt(qq, 10) + "] "
}

func readTimeSheets() error {
	file, err := getFile("TimeSheets.json")
	if err != nil {
		return err
	}
	timeSheetsdata, err := ioutil.ReadFile(file)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	err = json.Unmarshal(timeSheetsdata, &timeSheets)
	if err != nil {
		return err
	}
	return nil
}

func saveTimeSheets() {
	timeSheetsdata, err := json.MarshalIndent(timeSheets, "", "\t")
	if err != nil {
		printErr(err)
	}
	file, err := getFile("TimeSheets.json")
	if err != nil {
		printErr(err)
	}
	err = ioutil.WriteFile(file, timeSheetsdata, 0666)
	if err != nil {
		printErr(err)
	}
}

/**
 * 全局变量&数据类型的声明/定义
 * var CONFIG ConfigStruct  => 全局变量: 用户配置
 * type ConfigStruct struct => struct: 用户配置
 */

var AT_ME string
var CONFIG ConfigStruct
var timeSheets = make(map[int64]TimeSheetStruct)

type ConfigStruct struct {
	MasterQQ int64 `json:"master_qq"`
	QQGroup  int64 `json:"qq_group_num"`
}

type TimeSheetStruct struct {
	ShiftEnds string `json:"shift_ends"`
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

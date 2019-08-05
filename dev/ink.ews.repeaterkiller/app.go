package main

import (
	"encoding/json"
	"fmt"
	"github.com/Tnze/CoolQ-Golang-SDK/cqp"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

//go:generate cqcfg .
// cqp: 名称: 复读机杀手-Beta
// cqp: 版本: 0.9.1:2
// cqp: 作者: Rakuyo(imwhtl@gmail.com)
// cqp: 简介: 多磨，在下，复读机杀手desu。复读机，该杀。[/help 复读机]查看帮助
func main() {}
func init() {
	cqp.AppID = "ink.ews.repeaterkiller"
	cqp.Enable = onEnable
	cqp.Disable = onDisable
	cqp.GroupMsg = onGroupMsg
}

/**
 * 绑定cq事件函数
 * func onEnable() int32   => 启用插件
 * func onGroupMsg() int32 => 群消息事件
 * func onDisable() int32  => 停用插件 //TODO 未生效
 */

func onEnable() int32 {
	defer handleErr()
	cqp.AddLog(cqp.Debug, "启用插件", "插件启用: 复读机杀手(ink.ews.repeaterkiller)")
	applyConfig()
	//startPolling(5)
	return 0
}

func onGroupMsg(subType, msgID int32, fromGroup, fromQQ int64, fromAnonymous, msg string, font int32) int32 {
	defer handleErr()
	if killRepeater(fromGroup, fromQQ, msg) {
		LastMsg = msg
		NotRepeated = true
		RepeatTimes = 0
		DeathList = make(map[int]int64)
		return 1
	}
	reply := repeatMsg(fromQQ, msg)
	if DeathList[1] > 0 {
		cqp.AddLog(cqp.Debug, "死亡名单", fmt.Sprint(DeathList))
	}
	if reply != "" {
		cqp.SendGroupMsg(fromGroup, reply)
		return 1 /* 待定: 返回1则截停消息 返回0则仅复读 */
	}
	setResult := dealMsg(fromQQ, msg)
	if setResult != "" {
		cqp.SendGroupMsg(fromGroup, setResult)
		return 1
	}
	return 0
}

func onDisable() int32 {
	defer handleErr()
	saveConfig()
	cqp.AddLog(cqp.Debug, "停用插件", "插件停用: 复读机杀手(ink.ews.repeaterkiller)") /* BUG: 并不会显示 */
	return 0
}

/**
 * 主要业务逻辑
 * func killRepeater() => 判断是否拉黑消息发送者
 * func repeatMsg()    => 判断是否复读消息
 * func dealMsg()      => 处理指令
 */

func killRepeater(fromGroup, fromQQ int64, msg string) bool {
	if CONFIG.KillerTrigger > 0 && CONFIG.KillerTrigger == RepeatTimes {
		var dead_one int64
		var death_note string
		if (CONFIG.KillerTarget > 0) {
			dead_one = DeathList[CONFIG.KillerTarget]
			death_note = "根据规则，第" + strconv.Itoa(CONFIG.KillerTarget) + "位复读的选手" + atQQ(dead_one) +
				"将被禁言" + strconv.FormatInt(CONFIG.ReviveTime, 10) + "分钟。GGWP！"
		} else {
			dead_one = DeathList[rand.Intn(RepeatTimes)+1]
			death_note = "根据规则，随机抽选到一位幸运观众" + atQQ(dead_one) + "禁言" +
				strconv.FormatInt(CONFIG.ReviveTime, 10) + "分钟，让我们恭喜他。大吉大利，今晚口球！"
		}
		cqp.SendGroupMsg(fromGroup, death_note)
		if dead_one == cqp.GetLoginQQ() {
			//cqp.SendGroupMsg(fromGroup, cqp.GetLoginNick()+"免疫掉了这次禁言。")
			cqp.SendGroupMsg(fromGroup, "本机机人作为高贵的狗管理单方面宣布此次禁言无效。")
		} else {
			//cqp.SendGroupMsg(fromGroup, "(测试中，未真正禁言)")
			cqp.SetGroupBan(fromGroup, dead_one, CONFIG.ReviveTime*60)
		}
		return true
	}
	return false
}

func repeatMsg(fromQQ int64, msg string) string {
	var reply string
	if msg == LastMsg {
		RepeatTimes += 1
		DeathList[RepeatTimes] = fromQQ
	} else {
		NotRepeated = true
		RepeatTimes = 0
		DeathList = make(map[int]int64)
	}
	LastMsg = msg
	if CONFIG.RepeaterTrigger > 0 && RepeatTimes == CONFIG.RepeaterTrigger && NotRepeated {
		reply = msg
		NotRepeated = false
		RepeatTimes += 1
		DeathList[RepeatTimes] = cqp.GetLoginQQ()
	}
	//cqp.AddLog(cqp.Debug, "repeatMsg()", "CONFIG.RepeaterTrigger="+fmt.Sprint(CONFIG.RepeaterTrigger)+"    RepeatTimes="+fmt.Sprint(RepeatTimes)+"    LastMsg="+LastMsg)
	return reply
}

func dealMsg(fromQQ int64, msg string) string {
	var result string
	if strings.HasPrefix(msg, "/help 复读机") {
		var rules string
		if CONFIG.RepeaterTrigger == 0 {
			rules += "复读机已关闭。\n"
		} else {
			rules += "复读机已设置为：从第" + strconv.Itoa(CONFIG.RepeaterTrigger) + "条开始复读。\n"
		}
		if CONFIG.KillerTrigger == 0 {
			rules += "复读机杀手已关闭。\n"
		} else {
			rules += "复读机杀手已设置为：复读" + strconv.Itoa(CONFIG.KillerTrigger) + "次后开始狩猎。\n"
			if CONFIG.KillerTarget == 0 {
				rules += "复读机杀手已设置为：随机禁言一位幸运儿。\n"
			} else {
				rules += "复读机杀手已设置为：禁言第" + strconv.Itoa(CONFIG.KillerTarget) + "个复读机。\n"
			}
			rules += "已将禁言时间设置为" + strconv.FormatInt(CONFIG.ReviveTime, 10) + "分钟。"
		}
		rules += "======================================\n" +
			"/set 复读机 [触发复读所需的复读次数]\n" +
			"/set 复读机杀手 [触发禁言所需的复读次数]\n" +
			"/set 复读机杀谁 [触发禁言后的禁言目标] (为0时则随机选择目标)\n" +
			"/set 复读机复活时间 [禁言时间] (单位为分钟)"
		return rules
	}
	if CONFIG.MasterQQ != 0 && CONFIG.MasterQQ != fromQQ {
		return result
	}
	if strings.HasPrefix(msg, "/set 复读机") {
		set_item := strings.TrimPrefix(msg, "/set 复读机")
		if strings.HasPrefix(set_item, " ") {
			set_item = strings.TrimPrefix(set_item, " ")
			item, _ := strconv.Atoi(set_item)
			CONFIG.RepeaterTrigger = item
			if item == 0 {
				result = "复读机已关闭。"
			} else {
				result = "复读机已设置为：从第" + set_item + "条开始复读。"
			}
		} else if strings.HasPrefix(set_item, "杀手 ") {
			set_item = strings.TrimPrefix(set_item, "杀手 ")
			item, _ := strconv.Atoi(set_item)
			CONFIG.KillerTrigger = item
			if item == 0 {
				result = "复读机杀手已关闭。"
			} else {
				result = "复读机杀手已设置为：复读" + set_item + "次后开始狩猎。"
			}
		} else if strings.HasPrefix(set_item, "杀谁 ") {
			set_item = strings.TrimPrefix(set_item, "杀谁 ")
			item, _ := strconv.Atoi(set_item)
			CONFIG.KillerTarget = item
			if item == 0 {
				result = "复读机杀手已设置为：随机禁言一位幸运儿。"
			} else {
				result = "复读机杀手已设置为：禁言第" + set_item + "个复读机。"
			}
		} else if strings.HasPrefix(set_item, "复活时间 ") {
			set_item = strings.TrimPrefix(set_item, "复活时间 ")
			item, _ := strconv.ParseInt(set_item, 10, 64)
			CONFIG.ReviveTime = item
			result = "已将禁言时间设置为" + set_item + "分钟。"
		}
		saveConfig()
	}
	return result
}

/**
 * 次要业务辅助
 * func atQQ() string => 根据QQ号生成@该用户的CQ语句
 */

func atQQ(qq int64) string {
	return "[CQ:at,qq=" + strconv.FormatInt(qq, 10) + "]"
}

/**
 * 全局变量&数据类型的声明/定义
 * var LastMsg string          => 全局变量: 上次消息内容(判断是否复读的凭证)
 * var NotRepeated bool        => 全局变量: 未复读标记
 * var RepeatTimes int         => 全局变量: 计数器(目前复读次数)
 * var DeathList map[int]int64 => 全局变量: 死亡名单(参与复读的复读机们)
 * var CONFIG ConfigStruct     => 全局变量: 用户配置
 * type ConfigStruct struct    => struct: 用户配置
 */

var LastMsg string
var NotRepeated bool = true
var RepeatTimes int = 0
var DeathList map[int]int64
var CONFIG ConfigStruct

type ConfigStruct struct {
	MasterQQ        int64 `json:"master_qq"`
	QQGroup         int64 `json:"qq_group_num"`
	RepeaterTrigger int   `json:"repeater_trigger"`
	KillerTrigger   int   `json:"killer_trigger"`
	KillerTarget    int   `json:"killer_target"`
	ReviveTime      int64 `json:"revive_time"`
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

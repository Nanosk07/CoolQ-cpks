package main

import (
	"encoding/json"
	"fmt"
	"github.com/Tnze/CoolQ-Golang-SDK/cqp"
	"github.com/goinggo/mapstructure"
	"github.com/rakuyo42/cpk-by-rakuyo/utils/testapi"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

//go:generate cqcfg .
// cqp: 名称: 地震警报-Alpha
// cqp: 版本: 0.7.1:1
// cqp: 作者: Rakuyo(imwhtl@gmail.com)
// cqp: 简介: 自动播报地震信息。使用[/eq 2019-07-26]格式的指令查询指定日期以来最近的一次地震。
func main() {}
func init() {
	//TODO json处理参考https://www.jianshu.com/p/f7f930152482
	cqp.AppID = "ink.ews.earthquakes"
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
	cqp.AddLog(cqp.Debug, "启用插件", "插件启用: 地震警报(ink.ews.earthquakes)")
	applyConfig()
	startPolling(5)
	return 0
}

func onGroupMsg(subType, msgID int32, fromGroup, fromQQ int64, fromAnonymous, msg string, font int32) int32 {
	defer handleErr()
	reply := dealMsg(msg)
	if reply != "" {
		cqp.SendGroupMsg(fromGroup, reply)
		return 1
	}
	return 0
}

func onPrivateMsg(subType, msgID int32, fromQQ int64, msg string, font int32) int32 {
	defer handleErr()
	reply := dealMsg(msg)
	if reply != "" {
		cqp.SendPrivateMsg(fromQQ, reply)
		return 1
	}
	return 0
}

func onDisable() int32 {
	defer handleErr()
	saveConfig()
	cqp.AddLog(cqp.Debug, "停用插件", "插件停用: 地震警报(ink.ews.earthquakes)") /* BUG: 并不会显示 */
	return 0
}

/**
 * 主要业务逻辑
 * func startPolling()                     => 并发调用匿名轮询函数
 * func checkTodayEarthquake()             => 轮询函数查询业务
 * func dealMsg() string                   => 处理消息
 * func getLastEarthquake() EarthquakeInfo => 根据消息查询业务
 */

func startPolling(every_minute time.Duration) {
	go func() {
		for {
			checkTodayEarthquake()
			ticker := time.NewTicker(every_minute * time.Minute)
			<-ticker.C
		}
	}()
}

func checkTodayEarthquake() {
	//在[酷Q主程序文件夹 /data/app/ink.ews.earthquakes/config.json]编辑相关配置。
	last_eq := getLastEarthquake("", "")
	if (last_eq.EPI_DEPTH > 0) && (last_eq.CATA_ID != CONFIG.LastEqCataId) {
		retInfo := "最新地震播报：" + generateReply(last_eq)
		CONFIG.LastEqCataId = last_eq.CATA_ID
		CONFIG.LastEqTime = last_eq.O_TIME
		CONFIG.LastEqUrl = "http://news.ceic.ac.cn/" + last_eq.NEW_DID + ".html"
		saveConfig()
		cqp.SendGroupMsg(CONFIG.QQGroup, retInfo)
	} else {
		cqp.AddLog(cqp.Debug, "定时任务", "没有轮询到最新地震信息")
	}
}

func dealMsg(msg string) string {
	var retInfo string
	if strings.HasPrefix(msg, "/eq ") {
		start_date := msg[4:14]
		loc, _ := time.LoadLocation("Local")
		select_time, _ := time.ParseInLocation("2006-01-02", start_date, loc)
		today_time, _ := time.ParseInLocation("2006-01-02", time.Now().Format("2006-01-02"), loc)
		//cqp.AddLog(cqp.Debug, "查询日期", fmt.Sprint(select_time))
		//cqp.AddLog(cqp.Debug, "今天日期", fmt.Sprint(today_time))
		if select_time.Unix() < time.Date(2013, time.Month(1), 1, 0, 0, 0, 0, loc).Unix() {
			return "醒醒，9102年了。"
		} else if select_time.Unix() > today_time.Unix() {
			return "未来人不要剧透。"
		}
		last_eq := getLastEarthquake(start_date, "")
		if (last_eq.EPI_DEPTH > 0) {
			retInfo = last_eq.id + "「" + start_date + "」以来（四川附近地区）最近一次地震：\n" + generateReply(last_eq)
			CONFIG.LastEqId = last_eq.id /* BUG: forever empty id */
		} else {
			retInfo = "没有查询到" + start_date + "以来的地震情况。"
		}
	}
	return retInfo
}

func getLastEarthquake(start_date, end_date string) EarthquakeInfo {
	resp_json_str := testapi.RequestAPI("get", generateURL(start_date, end_date), "")
	resp_json_str = strings.TrimSuffix(strings.TrimPrefix(resp_json_str, "("), ")")
	resp_map := testapi.JsonToMap(resp_json_str)
	resp_slice := resp_map["shuju"]
	var last_eq_details OriginalEarthquakeJson
	var last_eq EarthquakeInfo
	for _, v := range resp_slice.([]interface{}) {
		if err := mapstructure.Decode(v, &last_eq_details); err != nil {
			printErr(err)
		} else {
			EarthquakeInfoMap := map[string]interface{}{
				"id":         "",
				"CATA_ID":    "",
				"O_TIME":     "",
				"LOCATION_C": "",
				"M":          "",
				"EPI_DEPTH":  "",
				"EPI_LON":    "",
				"EPI_LAT":    "",
				"NEW_DID":    ""}
			for k, v := range v.(map[string]interface{}) {
				for field, _ := range EarthquakeInfoMap {
					if field == k {
						EarthquakeInfoMap[k] = v
					}
				}
			}
			if err := mapstructure.Decode(EarthquakeInfoMap, &last_eq); err != nil {
				printErr(err)
			}
		}
		break
	}
	return last_eq
}

/**
 * 次要业务辅助
 * func generateReply() string => 根据地震数据生成用户友好的格式
 * func generateURL() string   => 根据用户配置生成请求数据接口的URL(GET方式带参)
 */

func generateReply(earthquake EarthquakeInfo) string {
	var longitude_str string
	var latitude_str string
	if strings.HasPrefix(earthquake.EPI_LON, "-") {
		longitude_str = "西经" + earthquake.EPI_LON
	} else {
		longitude_str = "东经" + strings.Trim(earthquake.EPI_LON, "-")
	}
	if strings.HasPrefix(earthquake.EPI_LAT, "-") {
		latitude_str = "南纬" + earthquake.EPI_LAT
	} else {
		latitude_str = "北纬" + strings.Trim(earthquake.EPI_LAT, "-")
	}
	retInfo := earthquake.LOCATION_C + "（" + longitude_str + "° " + latitude_str + "°）\n" +
		"于「" + earthquake.O_TIME + "」发生了「" + earthquake.M + "级地震（深度" +
		strconv.FormatFloat(earthquake.EPI_DEPTH, 'f', -1, 64) + "km）」\n" +
		"详情请点击http://news.ceic.ac.cn/" + earthquake.NEW_DID + ".html"
	return retInfo
}

func generateURL(start_date, end_date string) string {
	url := "http://www.ceic.ac.cn/ajax/search?"

	if CONFIG.Page > 0 {
		url += "page=" + strconv.Itoa(CONFIG.Page)
	} else {
		url += "page=1"
	}

	if start_date != "" {
		url += "&start=" + start_date
	} else {
		url += "&start=" + time.Now().Format("2006-01-02")
	}
	if end_date != "" {
		url += "&end" + end_date
	}

	if CONFIG.MinLongitude == 0 {
		url += "&jingdu1=95"
	} else if CONFIG.MinLongitude < 180 && CONFIG.MinLongitude > -180 {
		url += "&jingdu1=" + strconv.FormatFloat(CONFIG.MinLongitude, 'f', -1, 64)
	} else {
		url += "&jingdu1=95"
	}
	if CONFIG.MaxLongitude == 0 {
		url += "&jingdu2=110"
	} else if CONFIG.MaxLongitude < 180 && CONFIG.MaxLongitude > -180 {
		url += "&jingdu2=" + strconv.FormatFloat(CONFIG.MaxLongitude, 'f', -1, 64)
	} else {
		url += "&jingdu2=110"
	}

	if CONFIG.MinLatitude == 0 {
		url += "&weidu1=25"
	} else if CONFIG.MinLatitude < 90 && CONFIG.MinLatitude > -90 {
		url += "&weidu1=" + strconv.FormatFloat(CONFIG.MinLatitude, 'f', -1, 64)
	} else {
		url += "&weidu1=25"
	}
	if CONFIG.MaxLatitude == 0 {
		url += "&weidu2=35"
	} else if CONFIG.MaxLatitude < 90 && CONFIG.MaxLatitude > -90 {
		url += "&weidu2=" + strconv.FormatFloat(CONFIG.MaxLatitude, 'f', -1, 64)
	} else {
		url += "&weidu2=35"
	}

	if CONFIG.MinDepth != 0 {
		url += "&height1=" + strconv.FormatFloat(CONFIG.MinDepth, 'f', -1, 64)
	}
	if CONFIG.MaxDepth != 0 {
		url += "&height2=" + strconv.FormatFloat(CONFIG.MaxDepth, 'f', -1, 64)
	} else {
		url += "&height2"
	}

	if CONFIG.MinMagnitude != 0 {
		url += "&zhenji1=" + strconv.FormatFloat(CONFIG.MinMagnitude, 'f', -1, 64)
	}
	if CONFIG.MaxMagnitude != 0 {
		url += "&zhenji2=" + strconv.FormatFloat(CONFIG.MaxMagnitude, 'f', -1, 64)
	} else {
		url += "&zhenji2"
	}

	return url
}

/**
 * 全局变量&数据类型的声明/定义
 * var CONFIG ConfigStruct    => 全局变量: 用户配置
 * type ConfigStruct struct   => struct: 用户配置
 * type EarthquakeInfo struct => struct: 地震数据
 * type OriginalEarthquakeJson struct => struct: 地震数据原始Json
 */

var CONFIG ConfigStruct

type ConfigStruct struct {
	QQGroup      int64   `json:"qq_group_num"`
	LastEqId     string  `json:"last_earthquake_id,omitempty"`
	LastEqCataId string  `json:"last_earthquake_cataid"`
	LastEqTime   string  `json:"last_earthquake_time"`
	LastEqUrl    string  `json:"last_earthquake_url"`
	Page         int     `json:"query_page,omitempty"`
	StartDate    string  `json:"query_start_date,omitempty"`
	EndDate      string  `json:"query_end_date,omitempty"`
	MinLongitude float64 `json:"query_longitude_min"`
	MaxLongitude float64 `json:"query_longitude_max"`
	MinLatitude  float64 `json:"query_latitude_min"`
	MaxLatitude  float64 `json:"query_latitude_max"`
	MinDepth     float64 `json:"query_depth_min"`
	MaxDepth     float64 `json:"query_depth_max"`
	MinMagnitude float64 `json:"query_magnitude_min"`
	MaxMagnitude float64 `json:"query_magnitude_max"`
}

type EarthquakeInfo struct {
	id         string
	CATA_ID    string
	O_TIME     string
	LOCATION_C string
	M          string
	EPI_DEPTH  float64
	EPI_LON    string
	EPI_LAT    string
	NEW_DID    string
}

type OriginalEarthquakeJson struct {
	id           string
	CATA_ID      string
	SAVE_TIME    string
	O_TIME       string
	EPI_LAT      string
	EPI_LON      string
	EPI_DEPTH    float64
	AUTO_FLAG    string
	EQ_TYPE      string
	O_TIME_FRA   string
	M            string
	M_MS         string
	M_MS7        string
	M_ML         string
	M_MB         string
	M_MB2        string
	SUM_STN      string
	LOC_STN      string
	LOCATION_C   string
	LOCATION_S   string
	CATA_TYPE    string
	SYNC_TIME    string
	IS_DEL       string
	EQ_CATA_TYPE string
	NEW_DID      string
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

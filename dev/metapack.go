package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	CopyPack()
}

func CopyPack() {
	CQ_ROOT, _ := os.Getwd()
	CQ_DEV_PATH := filepath.Join(CQ_ROOT, "dev/")
	CQ_CPKS_PATH := filepath.Join(CQ_ROOT, "cpks/")
	cpks, _ := GetAllFiles(CQ_DEV_PATH, ".cpk")
	for _, cpk := range cpks {
		fmt.Printf("搜索到cpk文件 %s\n", cpk)
		file, err := os.Open(cpk)
		if err != nil {
			fmt.Println(err)
			return
		}
		bt, err := ioutil.ReadAll(file)
		if err != nil {
			fmt.Println(err)
			return
		}
		paths := strings.Split(cpk,"\\")
		CopyFile(bt, CQ_CPKS_PATH+"\\"+paths[len(paths)-1])
	}
}

//获取指定目录下的所有文件,包含子目录下的文件
func GetAllFiles(dirPth, suffix string) (files []string, err error) {
	var dirs []string
	dir, err := ioutil.ReadDir(dirPth)

	PthSep := string(os.PathSeparator)
	//suffix = strings.ToUpper(suffix) //忽略后缀匹配的大小写

	for _, fi := range dir {
		if fi.IsDir() { // 目录, 递归遍历
			dirs = append(dirs, dirPth+PthSep+fi.Name())
			GetAllFiles(dirPth+PthSep+fi.Name(), suffix)
		} else {
			// 过滤指定格式
			ok := strings.HasSuffix(fi.Name(), suffix)
			if ok {
				files = append(files, dirPth+PthSep+fi.Name())
			}
		}
	}

	// 读取子目录下文件
	for _, table := range dirs {
		temp, _ := GetAllFiles(table, suffix)
		for _, temp1 := range temp {
			files = append(files, temp1)
		}
	}
	return files, nil
}

func CopyFile(byte []byte, dst string) {
	dstFile, err := os.Create(dst)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer dstFile.Close()
	if _, err := io.Copy(dstFile, bytes.NewReader(byte)); err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("已复制到",dst)
	}
}

func dealComment(comm string) { //处理cqp注释
	switch {
	case strings.HasPrefix(comm, "// cqp: 名称:"):
		//if _, err := fmt.Sscanf(comm, "// cqp: 名称:%s", &info.Name); err != nil {
		//	log.Fatal("无法解析应用名称:", err)
		//}
	case strings.HasPrefix(comm, "// cqp: 版本:"):
		//var v1, v2, v3, seq int
		//if _, err := fmt.Sscanf(comm, "// cqp: 版本:%d.%d.%d:%d", &v1, &v2, &v3, &seq); err != nil {
		//	log.Fatal("无法解析版本号:", err)
		//}
	case strings.HasPrefix(comm, "// cqp: 作者:"):
		//if _, err := fmt.Sscanf(comm, "// cqp: 作者:%s", &info.Author); err != nil {
		//	log.Fatal("无法解析作者名:", err)
		//}
	case strings.HasPrefix(comm, "// cqp: 简介: "):
	}
}

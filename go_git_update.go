// go_git_update
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

const (
	Major_Ver = "1.1"
)

var (
	wg              *sync.WaitGroup
	b_mt            *bool
	s_home_rootpath *string
)

//判断文件或文件夹是否存在
func Exist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

func execCommand(commandName string, params []string, Dir_env string) bool {
	if *b_mt {
		defer wg.Done()
	}
	cmd := exec.Command(commandName, params...)
	cmd.Dir = Dir_env
	//显示运行的命令
	//fmt.Println(cmd.Args)
	if !*b_mt {
		fmt.Println("Dir:", Dir_env)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
		return false
	}
	cmd.Start()
	reader := bufio.NewReader(stdout)
	var out_buff bytes.Buffer
	//实时循环读取输出流中的一行内容
	for {
		line, err2 := reader.ReadString('\n')
		if err2 != nil || io.EOF == err2 {
			break
		}
		out_buff.WriteString(line)
	}
	if *b_mt {
		fmt.Println("Dir:", Dir_env)
	}
	fmt.Println(out_buff.String())
	cmd.Wait()
	return true
}

func git_Update_byDir(s_rootPath string) {
	folderList, err := ioutil.ReadDir(s_rootPath)
	if err != nil {
		fmt.Println("ioutil.ReadDir fail!")
	}
	s_PathSeparator := string(os.PathSeparator)
	for _, vFile := range folderList {
		if vFile.IsDir() {
			s_gitFolder := s_rootPath + s_PathSeparator + vFile.Name() + s_PathSeparator + ".git"
			if Exist(s_gitFolder) {
				if *b_mt {
					wg.Add(1)
					go execCommand("git", []string{"pull"}, s_rootPath+s_PathSeparator+vFile.Name())
				} else {
					execCommand("git", []string{"pull"}, s_rootPath+s_PathSeparator+vFile.Name())
				}
			} else {
				git_Update_byDir(s_rootPath + s_PathSeparator + vFile.Name())
			}
		} else {
			continue
		}
	}
}

func Init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s, Version: %s\n", os.Args[0], Major_Ver)
		fmt.Println("git batch update(pull)! by K.o.s[vbz276@gmail.com]!")
	}
	b_mt = flag.Bool("mt", false, "enable Multithreading")
	//home_rootpath := `F:\GoPortWin\go\src`
	s_home_rootpath = flag.String("dir", filepath.Dir(os.Args[0]), "Set Home RootPath")
}

func main() {
	Init()
	flag.Parse()
	//
	if *b_mt {
		wg = new(sync.WaitGroup)
		git_Update_byDir(*s_home_rootpath)
		wg.Wait()
	} else {
		git_Update_byDir(*s_home_rootpath)
	}

}

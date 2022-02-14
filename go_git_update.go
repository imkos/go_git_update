// go_git_update
package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"golang.org/x/sync/errgroup"
)

const (
	Major_Ver               = "2.1"
	DEFAULT_MAX_CHILD_TASKS = 20
	s_PathSeparator         = string(os.PathSeparator)
)

var (
	wg                *sync.WaitGroup
	b_mt              bool
	s_home_rootpath   string
	i_max_child_tasks uint
	needGitReset      bool
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s, Version: %s\n", os.Args[0], Major_Ver)
		fmt.Println("git batch update(pull)! by K.o.s[vbz276@gmail.com]!")
		flag.PrintDefaults()
	}
	flag.BoolVar(&b_mt, "mt", true, "enable Multithreading")
	flag.UintVar(&i_max_child_tasks, "ctasks", DEFAULT_MAX_CHILD_TASKS, "max Child tasks 1~20")
	// home_rootpath := `F:\GoPortWin\go\src`
	flag.StringVar(&s_home_rootpath, "dir", filepath.Dir(os.Args[0]), "Set Home RootPath")
	flag.BoolVar(&needGitReset, "gr", false, "is need git reset")
}

// 判断文件或文件夹是否存在
func Exist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

func execCommand(commandName string, params []string, Dir_env string) bool {
	if b_mt {
		if wg != nil {
			defer wg.Done()
		}
	}
	cmd := exec.Command(commandName, params...)
	cmd.Dir = Dir_env
	// 显示运行的命令
	// fmt.Println(cmd.Args)
	if !b_mt {
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
	// 实时循环读取输出流中的一行内容
	for {
		line, err2 := reader.ReadString('\n')
		if err2 != nil || io.EOF == err2 {
			break
		}
		out_buff.WriteString(line)
	}
	if b_mt {
		fmt.Println("Dir:", Dir_env)
	}
	fmt.Println(out_buff.String())
	cmd.Wait()
	return true
}

const (
	git_reset_pull = "git reset --hard && git pull"
	git_pull       = "git pull"
)

var gitCmd string

func git_Update_byDir(s_rootPath string, ch chan struct{}) {
	folderList, err := ioutil.ReadDir(s_rootPath)
	if err != nil {
		fmt.Println("ioutil.ReadDir fail!")
	}
	git_pull := func(spath string) {
		execCommand("git", []string{"pull"}, spath)
		if b_mt {
			<-ch
		}
	}
	for _, vFile := range folderList {
		if vFile.IsDir() {
			s_gitFolder := s_rootPath + s_PathSeparator + vFile.Name() + s_PathSeparator + ".git"
			if Exist(s_gitFolder) {
				if b_mt {
					wg.Add(1)
					// 在此处阻塞比在git_pull中阻塞要好一些
					ch <- struct{}{}
					go git_pull(s_rootPath + s_PathSeparator + vFile.Name())
				} else {
					git_pull(s_rootPath + s_PathSeparator + vFile.Name())
				}
			} else {
				git_Update_byDir(s_rootPath+s_PathSeparator+vFile.Name(), ch)
			}
		} else {
			continue
		}
	}
}

type result struct {
	path string
	bo   bool
}

func (r *result) print() {
	// fmt.Printf("path:%s, git update result: %v\n", r.path, r.bo)
}

func git_Update_byDir2(root string) {
	g, ctx := errgroup.WithContext(context.TODO())
	paths := make(chan string)

	g.Go(func() error {
		defer close(paths)
		return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				s_gitFolder := path + s_PathSeparator + ".git"
				if !Exist(s_gitFolder) {
					return nil
				}
			} else {
				return nil
			}
			select {
			case paths <- path:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		})
	})

	c := make(chan *result)
	numDigesters := int(i_max_child_tasks)
	for i := 0; i < numDigesters; i++ {
		g.Go(func() error {
			for po := range paths {
				select {
				case c <- &result{
					path: po,
					bo:   execCommand("bash", []string{"-c", gitCmd}, po),
				}:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		})
	}
	go func() {
		g.Wait()
		close(c)
	}()

	for r := range c {
		r.print()
	}
	if err := g.Wait(); err != nil {
		fmt.Println("g.Wait.error:", err)
		return
	}
}

func dirsWalk(s_rootPath string, ch chan string) error {
	folderList, err := ioutil.ReadDir(s_rootPath)
	if err != nil {
		fmt.Println("dirsWalk.ioutil.ReadDir fail!")
		return err
	}
	for _, vFile := range folderList {
		if vFile.IsDir() {
			s_gitFolder := s_rootPath + s_PathSeparator + vFile.Name() + s_PathSeparator + ".git"
			if Exist(s_gitFolder) {
				ch <- s_rootPath + s_PathSeparator + vFile.Name()
			} else {
				dirsWalk(s_rootPath+s_PathSeparator+vFile.Name(), ch)
			}
		} else {
			continue
		}
	}
	return nil
}

func git_Update_byDir3(root string) {
	g, ctx := errgroup.WithContext(context.TODO())
	paths := make(chan string)

	g.Go(func() error {
		defer close(paths)
		return dirsWalk(root, paths)
	})

	c := make(chan *result)
	numDigesters := int(i_max_child_tasks)
	for i := 0; i < numDigesters; i++ {
		g.Go(func() error {
			for po := range paths {
				select {
				case c <- &result{
					path: po,
					bo:   execCommand("bash", []string{"-c", gitCmd}, po),
				}:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		})
	}
	go func() {
		g.Wait()
		close(c)
	}()

	for r := range c {
		r.print()
	}
	if err := g.Wait(); err != nil {
		fmt.Println("g.Wait.error:", err)
		return
	}
}

func main() {
	flag.Parse()
	// cap: 1~30
	if i_max_child_tasks == 0 || i_max_child_tasks > DEFAULT_MAX_CHILD_TASKS {
		i_max_child_tasks = DEFAULT_MAX_CHILD_TASKS
	}
	ch_max_exec := make(chan struct{}, i_max_child_tasks)
	if needGitReset {
		gitCmd = git_reset_pull
	} else {
		gitCmd = git_pull
	}
	//
	if b_mt {
		/* old方法
		wg = new(sync.WaitGroup)
		git_Update_byDir(s_home_rootpath, ch_max_exec)
		wg.Wait()
		*/
		git_Update_byDir3(s_home_rootpath)
	} else {
		git_Update_byDir(s_home_rootpath, ch_max_exec)
	}
}

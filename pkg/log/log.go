package _log

import (
	"errors"
	"fmt"
	"github.com/dataznGao/bingo"
	"github.com/dataznGao/leo/constant"
	"github.com/dataznGao/leo/pkg/caller"
	"github.com/dataznGao/leo/pkg/callgraph"
	_ast "github.com/dataznGao/leo/pkg/log/ast"
	"github.com/dataznGao/leo/util"
	"github.com/dataznGao/leo/util/task"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

var threshold = 10

func Log(inputPath, outputPath string) error {
	// 1. 启动服务端
	go func() { caller.StartServe() }()
	// 2. 找到所有的测试文件
	testPath, err := util.LoadTestPath(inputPath)
	if err != nil {
		return err
	}
	allDiffs := make([]*callgraph.Diff, 0)

	if len(testPath) < threshold {
		threshold = len(testPath)
	}
	// 3. 进行diff图生成
	cnt := 0
	for _, s := range testPath {
		if diffs, err := generateDiff(inputPath, s, outputPath); err != nil {
			log.Printf("[leo] WARN testPath: %v run has err: %v\n", s, err)
		} else {
			allDiffs = append(allDiffs, diffs...)
			cnt++
			if cnt == threshold {
				break
			}
		}
	}
	allDiffs = callgraph.DedupDiff(allDiffs)
	// 4. 根据diff图打日志
	return DiffLog(inputPath, outputPath, allDiffs)
}

func DiffLog(inputPath, outputPath string, diffs []*callgraph.Diff) error {
	files, notGoFiles, err := LoadPackage(inputPath)
	if err != nil {
		return err
	}
	// 注入error, 并产生import log
	for k, file := range files {
		code, hasLogged := _ast.InjureLog(k, file, diffs)
		err := util.CreateFile(util.CompareAndExchange(k, outputPath, inputPath), code)
		if err != nil {
			return err
		}
		file.Logged = hasLogged
	}
	//xlsxInfo := make([][]string, 0)
	//xlsxInfo = append(xlsxInfo, []string{"filepath", "caller", "callee"})
	//for _, diff := range diffs {
	//	xlsxInfo = append(xlsxInfo, diff.PrintTrace())
	//}
	//util.DataToExcel("/Users/misery/GolandProjects/leo/ConditionInversedFault.xlsx", xlsxInfo)
	return fillPackage(files, notGoFiles, outputPath, inputPath)
}

// generateDiff inputPath: 项目文件夹所在的地址, testPath: 单元测试所在的文件夹地址, outputPath: 日志增强后所在的地址
func generateDiff(inputPath, testPath, outputPath string) ([]*callgraph.Diff, error) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Recovered from:", err)
		}
	}()
	// 校验
	// a. 结尾校验
	if strings.HasSuffix(outputPath, constant.Separator) {
		outputPath = outputPath[:len(outputPath)-1]
	}
	// b. inputPath, testPath校验
	if !strings.HasPrefix(testPath, inputPath) {
		return nil, errors.New("[bingo] the testPath or inputPath set err! please check! err")
	}
	// leo/scene1.test.init
	index := strings.LastIndex(inputPath, constant.Separator)
	removeDir := inputPath[:index] + constant.Separator + "leo_tmp"
	// 保证临时文件夹会被删除
	os.RemoveAll(removeDir)
	defer os.RemoveAll(removeDir)
	group := task.NewGroup(2)
	rawCallGraph := make(map[string]map[string]string, 0)
	modCallGraph := make(map[string]map[string]string, 0)
	var err error

	// 1. 原始调用图生成
	group.Add(func() {
		myTestPath := testPath
		log.Printf("[leo] INFO ===== 原始调用图生成开始 =====")
		log.Printf("[leo] INFO ===== 动态原始调用图生成开始 =====")
		// 0. 对代码进行插桩
		_index := strings.LastIndex(inputPath, constant.Separator)
		realInputPath := inputPath[:_index] + constant.Separator + constant.EnhanceInputPath
		defer os.RemoveAll(realInputPath)
		// 测试文件的相对位置改变
		myTestPath = util.CompareAndExchange(myTestPath, realInputPath, inputPath)
		// 原始调用图用0标识
		num := 0
		err = InsertCollector(inputPath, realInputPath, num)
		if err != nil {
			log.Printf("[leo] ERROR ===== 动态调用图插桩失败 =====")
		}
		inputPath = realInputPath
		//dyRawCallGraph, err := callgraph.DynamicAnal(inputPath, myTestPath, num)
		//if err != nil {
		//	log.Printf("[leo] ERROR ===== 动态调用图生成失败 =====")
		//}
		//println(dyRawCallGraph)
		log.Printf("[leo] INFO ===== 静态原始调用图生成开始 =====")
		rawCallGraph, err = callgraph.Anal(inputPath, myTestPath)
		if err != nil {
			log.Printf("[leo] WARN ===== 原始调用图生成失败 =====")
		}
		log.Printf("[leo] INFO ===== 原始调用图生成完毕 =====")
	})

	// 2. 调用故障注入，故障调用图生成
	group.Add(func() {
		myTestPath := testPath
		tmpPath := removeDir
		log.Printf("tmpPath: %v", tmpPath)
		// 进行故障注入
		env := bingo.CreateMutationEnv(inputPath, tmpPath, myTestPath)
		env.SyncFault("*.*.*.*")
		env.SwitchMissDefaultFault("*.*.*.*")
		env.ExceptionUncaughtFault("*.*.*.*")
		env.ExceptionShortcircuitFault("*.*.*.*")
		env.ExceptionUnhandledFault("*.*.*.*")
		f := bingo.MutationPerformer{}
		log.Printf("[leo] INFO ===== 故障注入启动 =====")
		err := f.SetEnv(env).Run(true)
		if err != nil {
			log.Fatal("fault inject err !")
		}
		log.Printf("[leo] INFO ===== 故障注入完毕 =====")
		log.Printf("[leo] INFO ===== 故障调用图生成开始 =====")
		log.Printf("[leo] INFO ===== 原始调用图生成开始 =====")
		log.Printf("[leo] INFO ===== 动态原始调用图生成开始 =====")
		// 0. 对代码进行插桩
		_index := strings.LastIndex(tmpPath, constant.Separator)
		realInputPath := tmpPath[:_index] + constant.Separator + constant.TmpEnhanceInputPath
		defer os.RemoveAll(realInputPath)
		// 测试文件的相对位置改变
		myTestPath = util.CompareAndExchange(myTestPath, realInputPath, inputPath)
		// 故障调用图用1标识
		num := 1
		err = InsertCollector(tmpPath, realInputPath, num)
		if err != nil {
			log.Printf("[leo] ERROR ===== 动态调用图插桩失败 =====")
		}
		tmpPath = realInputPath
		dyModCallGraph, err := callgraph.DynamicAnal(tmpPath, myTestPath, num)
		println(dyModCallGraph)
		modCallGraph, err = callgraph.Anal(tmpPath, myTestPath)
		if err != nil {
			log.Printf("[leo] WARN ===== 故障调用图生成失败 =====")
		}
		log.Printf("[leo] INFO ===== 故障调用图生成完毕 =====")
	})
	group.Start()
	group.Wait()
	if err != nil {
		return nil, err
	}
	log.Printf("[leo] INFO 开始比对调用图")
	diffs := callgraph.Compare(rawCallGraph, modCallGraph, inputPath)
	// /Users/misery/GolandProjects/rpc_demo/tttt/aaas MyT RunClient1
	// /Users/misery/GolandProjects/rpc_demo/tttt/aaas MyT RunClient1$1
	log.Printf("[leo] INFO 共有%v个diff", len(diffs))
	log.Printf("[leo] INFO 调用图比对完成")
	return diffs, nil
}

func InsertCollector(inputPath, outputPath string, num int) error {
	files, notGoFiles, err := LoadPackage(inputPath)
	if err != nil {
		return err
	}
	// 插桩, 产生import leo
	for k, file := range files {
		if strings.HasSuffix(k, "_test.go") {
			continue
		}
		code := caller.StartCollect(file.File, num)
		err = util.CreateFile(util.CompareAndExchange(k, outputPath, inputPath), code)
		if err != nil {
			return err
		}
	}
	return fillPackage(files, notGoFiles, outputPath, inputPath)
}

func fillPackage(files map[string]*_ast.File, notGoFiles []string, outputPath, inputPath string) error {
	for k, v := range files {
		if !v.Logged {
			err := util.CreateFile(util.CompareAndExchange(k, outputPath, inputPath),
				util.GetFileCode(v.File))
			if err != nil {
				return err
			}
		}
	}
	for _, file := range notGoFiles {
		log.Printf("[bingo] INFO 填充输出包, 文件: %v", file)
		readFile, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}
		err = util.CreateFile(util.CompareAndExchange(file, outputPath, inputPath),
			readFile)
		if err != nil {
			return err
		}
	}
	return nil
}

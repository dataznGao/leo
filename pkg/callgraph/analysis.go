package callgraph

import (
	"errors"
	"fmt"
	"github.com/dataznGao/leo/util"
	"github.com/dataznGao/leo/util/task"
	"go/build"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/callgraph/static"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/pointer"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

type CallGraphType string

const (
	CallGraphTypeStatic  CallGraphType = "static"
	CallGraphTypeCha                   = "cha"
	CallGraphTypeRta                   = "rta"
	CallGraphTypePointer               = "pointer"
)

// ==[ type def/func: analysis   ]===============================================
type renderOpts struct {
	cacheDir string
	focus    string
	group    []string
	ignore   []string
	include  []string
	limit    []string
	nointer  bool
	refresh  bool
	nostd    bool
	algo     CallGraphType
}

// mainPackages returns the main packages to analyze.
// Each resulting package is named "main" and has a main function.
func mainPackages(pkgs []*ssa.Package) ([]*ssa.Package, error) {
	var mains []*ssa.Package
	for _, p := range pkgs {
		if p != nil && p.Pkg.Name() == "main" && p.Func("main") != nil {
			mains = append(mains, p)
		}
	}
	if len(mains) == 0 {
		return nil, fmt.Errorf("no main packages")
	}
	return mains, nil
}

// ==[ type def/func: analysis   ]===============================================
type analysis struct {
	opts        *renderOpts
	prog        *ssa.Program
	pkgs        []*ssa.Package
	mainPkg     *ssa.Package
	callgraph   *callgraph.Graph
	packageName string
}

var Analysis *analysis

// DoAnalysis 转成调用图的主要方法
func (a *analysis) DoAnalysis(
	algo CallGraphType,
	dir string,
	tests bool,
	args []string,
) error {
	cfg := &packages.Config{
		Mode:       packages.LoadAllSyntax,
		Tests:      tests,
		Dir:        dir,
		BuildFlags: build.Default.BuildTags,
	}
	log.Printf("[leo] INFO 开始加载包, 目录为: %v", dir)
	initial, err := packages.Load(cfg, args...)
	if err != nil {
		return err
	}

	if packages.PrintErrors(initial) > 0 {
		return fmt.Errorf("packages contain errors")
	}
	log.Printf("[leo] INFO 加载包成功")
	// ssautil用来读取go源码并解析成相应的SSA中间代码
	a.prog, a.pkgs = ssautil.AllPackages(initial, 0)
	a.prog.Build()
	log.Printf("[leo] INFO 开始生成ssa中间码")
	var graph *callgraph.Graph
	var mainPkg *ssa.Package

	switch algo {
	case CallGraphTypeStatic:
		graph = static.CallGraph(a.prog)
	case CallGraphTypeCha:
		graph = cha.CallGraph(a.prog)
	case CallGraphTypeRta:
		mains, err := mainPackages(a.prog.AllPackages())
		if err != nil {
			return err
		}
		var roots []*ssa.Function
		mainPkg = mains[0]
		for _, main := range mains {
			roots = append(roots, main.Func("main"))
		}
		graph = rta.Analyze(roots, true).CallGraph
	case CallGraphTypePointer:
		mains, err := mainPackages(a.prog.AllPackages())
		if err != nil {
			return err
		}
		mainPkg = mains[0]
		config := &pointer.Config{
			Mains:          mains,
			BuildCallGraph: true,
		}
		log.Printf("[leo] INFO pointer开始分析ssa中间码")
		ptares, err := pointer.Analyze(config)
		if err != nil {
			return err
		}
		graph = ptares.CallGraph
		log.Printf("[leo] INFO pointer分析ssa中间码完毕")
	default:
		return fmt.Errorf("invalid call graph type: %s", a.opts.algo)
	}

	a.mainPkg = mainPkg
	a.callgraph = graph
	return nil
}

func (a *analysis) OptsSetup() {
	a.opts = &renderOpts{
		cacheDir: *cacheDir,
		focus:    *focusFlag,
		group:    []string{*groupFlag},
		ignore:   []string{*ignoreFlag},
		include:  []string{*includeFlag},
		limit:    []string{*limitFlag},
		nointer:  *nointerFlag,
		nostd:    *nostdFlag,
	}
}

func (a *analysis) ProcessListArgs() (e error) {
	var groupBy []string
	var ignorePaths []string
	var includePaths []string
	var limitPaths []string

	for _, g := range strings.Split(a.opts.group[0], ",") {
		g := strings.TrimSpace(g)
		if g == "" {
			continue
		}
		if g != "pkg" && g != "type" {
			e = errors.New("invalid group option")
			return
		}
		groupBy = append(groupBy, g)
	}

	for _, p := range strings.Split(a.opts.ignore[0], ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			ignorePaths = append(ignorePaths, p)
		}
	}

	for _, p := range strings.Split(a.opts.include[0], ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			includePaths = append(includePaths, p)
		}
	}

	for _, p := range strings.Split(a.opts.limit[0], ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			limitPaths = append(limitPaths, p)
		}
	}

	a.opts.group = groupBy
	a.opts.ignore = ignorePaths
	a.opts.include = includePaths
	a.opts.limit = limitPaths

	return
}

func (a *analysis) OverrideByHTTP(r *http.Request) {
	if f := r.FormValue("f"); f == "all" {
		a.opts.focus = ""
	} else if f != "" {
		a.opts.focus = f
	}
	if std := r.FormValue("std"); std != "" {
		a.opts.nostd = false
	}
	if inter := r.FormValue("nointer"); inter != "" {
		a.opts.nointer = true
	}
	if refresh := r.FormValue("refresh"); refresh != "" {
		a.opts.refresh = true
	}
	if g := r.FormValue("group"); g != "" {
		a.opts.group[0] = g
	}
	if l := r.FormValue("limit"); l != "" {
		a.opts.limit[0] = l
	}
	if ign := r.FormValue("ignore"); ign != "" {
		a.opts.ignore[0] = ign
	}
	if inc := r.FormValue("include"); inc != "" {
		a.opts.include[0] = inc
	}
	return
}

type SyncVertxList struct {
	vertxs []*Vertx
	mu     sync.Mutex
}

// Render basically do printOutput() with previously checking
// focus option and respective package
func (a *analysis) Render() ([]*Vertx, error) {
	var err error
	vertxs := make([]*Vertx, 0)
	smap := sync.Map{}
	// 分3个协程并发处理
	num := 3
	group := task.NewGroup(num)
	pack := a.prog.AllPackages()
	myPack := make([]*ssa.Package, 0)
	for _, p := range pack {
		if strings.Contains(p.Pkg.Path(), a.packageName) {
			myPack = append(myPack, p)
		}
	}
	n := len(myPack)
	batchSize := n / num
	log.Printf("[leo] INFO 多协程进行渲染, 协程数量: %v, 数据总量: %v, 数据batch size: %v",
		num, n, batchSize)
	outputMu := sync.Mutex{}
	for i := 0; i < num; i++ {
		// 单协程计算任务
		batch := make([]*ssa.Package, 0)
		if i == num-1 {
			batch = myPack[i*batchSize : n]
		} else {
			batch = myPack[i*batchSize : (i+1)*batchSize]
		}
		log.Printf("[leo] INFO go_routine_id: %v, 处理数量: %v", i+1, len(batch))
		CalRender := func() {
			for _, pkg := range batch {
				dot, err := printOutput(
					&outputMu,
					a.prog,
					a.mainPkg,
					a.callgraph,
					pkg.Pkg,
					a.opts.limit,
					a.opts.ignore,
					a.opts.include,
					a.opts.group,
					a.opts.nostd,
					a.opts.nointer,
				)
				if err != nil {
					log.Fatalf("pkg parse err, err: %v", err)
				}
				for s := range dot {
					smap.Store(s.ToString(), s)
				}
			}
		}
		group.Add(CalRender)
	}
	group.Start()
	group.Wait()
	if err != nil {
		return nil, fmt.Errorf("processing failed: %v", err)
	}
	smap.Range(func(key, value interface{}) bool {
		vertxs = append(vertxs, value.(*Vertx))
		return true
	})
	log.Printf("[leo] INFO 共获取到%v条调用边", len(vertxs))
	return vertxs, nil
}

func (a *analysis) FindCachedImg() string {
	if a.opts.cacheDir == "" || a.opts.refresh {
		return ""
	}

	focus := a.opts.focus
	if focus == "" {
		focus = "all"
	}
	focusFilePath := focus + "." + *outputFormat
	absFilePath := filepath.Join(a.opts.cacheDir, focusFilePath)

	if exists, err := pathExists(absFilePath); err != nil || !exists {
		log.Println("not cached img:", absFilePath)
		return ""
	}

	log.Println("hit cached img")
	return absFilePath
}

func (a *analysis) CacheImg(img string) error {
	if a.opts.cacheDir == "" || img == "" {
		return nil
	}

	focus := a.opts.focus
	if focus == "" {
		focus = "all"
	}
	absCacheDirPrefix := filepath.Join(a.opts.cacheDir, focus)
	absCacheDirPath := strings.TrimRightFunc(absCacheDirPrefix, func(r rune) bool {
		return r != '\\' && r != '/'
	})
	err := os.MkdirAll(absCacheDirPath, os.ModePerm)
	if err != nil {
		return err
	}

	absFilePath := absCacheDirPrefix + "." + *outputFormat
	_, err = copyFile(img, absFilePath)
	if err != nil {
		return err
	}

	return nil
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func copyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)

	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func Compare(a, b map[string]map[string]string, inputPath string) []*Diff {
	diff := make([]*Diff, 0)
	for caller, m := range a {
		for callee, descA := range m {
			callerPath := String2Func(strings.Replace(caller, util.GetPackageName(inputPath), inputPath, 1))
			calleePath := String2Func(strings.Replace(callee, util.GetPackageName(inputPath), inputPath, 1))
			if calleeMapB, ok := b[caller]; ok {
				if descB, ok := calleeMapB[callee]; ok {
					if descB == descA {
						continue
					} else {
						// case1: 描述不同
						diff = append(diff, &Diff{
							NodeA: &Node{
								Caller:      callerPath,
								Callee:      calleePath,
								Description: descA,
							},
							NodeB: &Node{
								Caller:      callerPath,
								Callee:      calleePath,
								Description: descB,
							},
							Detail: &DescDiff,
						})
					}
				} else {
					// case2: B中不存在对应的callee
					diff = append(diff, &Diff{
						NodeA: &Node{
							Caller:      callerPath,
							Callee:      calleePath,
							Description: descA,
						},
						Detail: &SideLackDiff,
					})
				}
			} else {
				// case3: B中不存在对应的caller
				diff = append(diff, &Diff{
					NodeA: &Node{
						Caller:      callerPath,
						Callee:      calleePath,
						Description: descA,
					},
					Detail: &SideLackDiff,
				})
			}
		}
	}

	for caller, m := range b {
		for callee, descB := range m {
			callerPath := String2Func(strings.Replace(caller, util.GetPackageName(inputPath), inputPath, 1))
			calleePath := String2Func(strings.Replace(callee, util.GetPackageName(inputPath), inputPath, 1))
			if calleeMapA, ok := a[caller]; ok {
				if descA, ok := calleeMapA[callee]; ok {
					if descB == descA {
						continue
					} else {
						// case1: 描述不同
						diff = append(diff, &Diff{
							NodeA: &Node{
								Caller:      callerPath,
								Callee:      calleePath,
								Description: descA,
							},
							NodeB: &Node{
								Caller:      callerPath,
								Callee:      calleePath,
								Description: descB,
							},
							Detail: &DescDiff,
						})
					}
				} else {
					// case2: A中不存在对应的callee
					diff = append(diff, &Diff{
						NodeB: &Node{
							Caller:      callerPath,
							Callee:      calleePath,
							Description: descB,
						},
						Detail: &SideLackDiff,
					})
				}
			} else {
				// case3: A中不存在对应的caller
				diff = append(diff, &Diff{
					NodeB: &Node{
						Caller:      callerPath,
						Callee:      calleePath,
						Description: descB,
					},
					Detail: &SideLackDiff,
				})
			}
		}
	}
	return DedupDiff(diff)
}

func DedupDiff(diff []*Diff) []*Diff {
	res := make([]*Diff, 0)
	deMap := make(map[string]*Diff, 0)
	for _, di := range diff {
		deMap[di.ToString()] = di
	}
	sortMap := make(map[string][]*Diff, 0)
	for _, d := range deMap {
		if _, ok := sortMap[d.NodeA.Caller.ToString()]; !ok {
			sortMap[d.NodeA.Caller.ToString()] = make([]*Diff, 0)
		}
		sortMap[d.NodeA.Caller.ToString()] = append(sortMap[d.NodeA.Caller.ToString()], d)

	}
	for _, diffs := range sortMap {
		for _, d := range diffs {
			res = append(res, d)
		}
	}
	return res
}

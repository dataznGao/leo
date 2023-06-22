// go-callvis: a tool to help visualize the call graph of a Go program.
package callgraph

import (
	"flag"
	"fmt"
	"go/build"
	"golang.org/x/tools/go/buildutil"
	"leo/util"
	"log"
)

var (
	focusFlag     = flag.String("focus", "main", "Focus specific package using name or import path.")
	groupFlag     = flag.String("group", "pkg", "Grouping functions by packages and/or types [pkg, type] (separated by comma)")
	limitFlag     = flag.String("limit", "", "Limit package paths to given prefixes (separated by comma)")
	ignoreFlag    = flag.String("ignore", "", "Ignore package paths containing given prefixes (separated by comma)")
	includeFlag   = flag.String("include", "", "Include package paths with given prefixes (separated by comma)")
	nostdFlag     = flag.Bool("nostd", false, "Omit calls to/from packages in standard library.")
	nointerFlag   = flag.Bool("nointer", false, "Omit calls to unexported functions.")
	testFlag      = flag.Bool("tests", false, "Include test code.")
	graphvizFlag  = flag.Bool("graphviz", false, "Use Graphviz's dot program to render images.")
	httpFlag      = flag.String("http", ":7878", "HTTP service address.")
	skipBrowser   = flag.Bool("skipbrowser", false, "Skip opening browser.")
	outputFile    = flag.String("file", "", "output filename - omit to use server mode")
	outputFormat  = flag.String("format", "svg", "output file format [svg | png | jpg | ...]")
	cacheDir      = flag.String("cacheDir", "", "Enable caching to avoid unnecessary re-rendering, you can force rendering by adding 'refresh=true' to the URL query or emptying the cache directory")
	callgraphAlgo = flag.String("algo", CallGraphTypePointer, fmt.Sprintf("The algorithm used to construct the call graph. Possible values inlcude: %q, %q, %q, %q",
		CallGraphTypeStatic, CallGraphTypeCha, CallGraphTypeRta, CallGraphTypePointer))

	debugFlag   = flag.Bool("debug", false, "Enable verbose logger.")
	versionFlag = flag.Bool("version", false, "Show version and exit.")
)

func init() {
	flag.Var((*buildutil.TagsFlag)(&build.Default.BuildTags), "tags", buildutil.TagsFlagDoc)
	// Graphviz options
	flag.UintVar(&minlen, "minlen", 2, "Minimum edge length (for wider output).")
	flag.Float64Var(&nodesep, "nodesep", 0.35, "Minimum space between two adjacent nodes in the same rank (for taller output).")
	flag.StringVar(&nodeshape, "nodeshape", "box", "graph node shape (see graphvis manpage for valid values)")
	flag.StringVar(&nodestyle, "nodestyle", "filled,rounded", "graph node style (see graphvis manpage for valid values)")
	flag.StringVar(&rankdir, "rankdir", "LR", "Direction of graph layout [LR | RL | TB | BT]")
}

func logf(f string, a ...interface{}) {
	if *debugFlag {
		log.Printf(f, a...)
	}
}

func Draw(isTest bool, testPath, inputPath, packageName string, nostd bool) ([]*Vertx, error) {
	args := []string{testPath}

	anal := new(analysis)
	log.Printf("[leo] INFO 开始进行调用图分析")
	if err := anal.DoAnalysis(CallGraphType(*callgraphAlgo), inputPath, isTest, args); err != nil {
		return nil, err
	}
	anal.OptsSetup()
	anal.opts.nostd = nostd
	anal.packageName = packageName

	// 将图转成可读的格式
	log.Printf("[leo] INFO 开始渲染分析图")
	output, _ := anal.Render()
	log.Printf("[leo] INFO 分析图渲染完毕")
	return output, nil
}

func Anal(inputPath, testPath string) (map[string]map[string]string, error) {
	packageName := util.GetPackageName(inputPath)
	exchange := util.CompareAndExchange(testPath, packageName, inputPath)
	vertxs, err := Draw(true, exchange, inputPath, packageName, true)
	if err != nil {
		return nil, err
	}
	nodes := make(map[string]interface{})
	for _, vertx := range vertxs {
		caller := vertx.Caller
		callee := vertx.Callee
		nodes[caller] = ""
		nodes[callee] = ""
	}
	res := make(map[string]map[string]string, len(nodes))
	for i := range nodes {
		res[i] = make(map[string]string)
	}
	for _, vertx := range vertxs {
		caller := vertx.Caller
		callee := vertx.Callee
		res[caller][callee] = vertx.Description
	}
	return res, nil
}

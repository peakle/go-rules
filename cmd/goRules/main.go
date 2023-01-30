package main

import (
	"bytes"
	"fmt"
	"go/token"
	"io/ioutil"
	"regexp"
	"strings"
	"sync"

	"github.com/quasilyte/go-ruleguard/ruleguard"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/peakle/go-rules/precompile/rulesdata"
)

var (
	flagDebug    bool
	flagDisable  string
	flagEnable   string
	flagRules    string
	flagSkipDirs string
)

// Version contains extra version info.
// It's initialized via ldflags -X when goRules is built with Make.
// Can contain a git hash (dev builds) or a version tag (release builds).
var Version string

func docString() string {
	doc := "lint go files by my custom checks"
	if Version == "" {
		return doc
	}
	return doc + " (" + Version + ")"
}

// Analyzer exports goRules as an analysis-compatible object.
var Analyzer = &analysis.Analyzer{
	Name: "goRules",
	Doc:  docString(),
	Run:  runAnalyzer,
}

var (
	globalEngineMu      sync.Mutex
	globalEngine        *ruleguard.Engine
	globalEngineErrored bool
	skipDirsPatterns    []*regexp.Regexp
)

func init() {
	Analyzer.Flags.BoolVar(&flagDebug, "V", false, "enable verbose mode")
	Analyzer.Flags.StringVar(&flagDisable, "disable", "", "comma-separated list of disabled groups or skip empty to enable everything: #perfomance,#experimental")
	Analyzer.Flags.StringVar(&flagEnable, "enable", "<all>", "comma-separated list of enabled groups or skip empty to enable everything: #diagnostic,#style")
	Analyzer.Flags.StringVar(&flagRules, "rules", "", "comma-separated list of rules files")
	Analyzer.Flags.StringVar(&flagSkipDirs, "skip-dirs", "", "comma-separated list of dirs for skip")
}

func prepareEngine() error {
	globalEngineMu.Lock()
	defer globalEngineMu.Unlock()

	if globalEngine != nil {
		return nil
	}

	if globalEngineErrored {
		return nil
	}

	if err := newEngine(); err != nil {
		globalEngineErrored = true
		return err
	}

	return nil
}

func newEngine() error {
	enabledGroups := make(map[string]bool)
	disabledGroups := make(map[string]bool)
	enabledTags := make(map[string]bool)
	disabledTags := make(map[string]bool)

	for _, g := range strings.Split(flagDisable, ",") {
		g = strings.TrimSpace(g)
		if t := strings.Split(g, "#"); len(t) == 2 {
			disabledTags[t[1]] = true
			continue
		}

		disabledGroups[g] = true
	}
	if flagEnable != "<all>" {
		for _, g := range strings.Split(flagEnable, ",") {
			g = strings.TrimSpace(g)
			if t := strings.Split(g, "#"); len(t) == 2 {
				enabledTags[t[1]] = true
				continue
			}

			enabledGroups[g] = true
		}
	}
	inEnabledTags := func(g *ruleguard.GoRuleGroup) bool {
		for _, t := range g.DocTags {
			if enabledTags[t] {
				return true
			}
		}
		return false
	}
	inDisabledTags := func(g *ruleguard.GoRuleGroup) string {
		for _, t := range g.DocTags {
			if disabledTags[t] {
				return t
			}
		}
		return ""
	}

	if flagDebug {
		debugPrint(fmt.Sprintf("enabled tags: %+v", enabledTags))
		debugPrint(fmt.Sprintf("disabled tags: %+v", disabledTags))
	}

	ctx := &ruleguard.LoadContext{
		DebugImports: flagDebug,
		Fset:         token.NewFileSet(),
		DebugPrint:   debugPrint,
		GroupFilter: func(g *ruleguard.GoRuleGroup) bool {
			whyDisabled := ""
			enabled := flagEnable == "<all>" || enabledGroups[g.Name] || inEnabledTags(g)

			switch {
			case !enabled:
				whyDisabled = "not enabled by name or tag (-enable flag)"
			case disabledGroups[g.Name]:
				whyDisabled = "disabled by name (-disable flag)"
			default:
				if tag := inDisabledTags(g); tag != "" {
					whyDisabled = fmt.Sprintf("disabled by %s tag (-disable flag)", tag)
				}
			}

			if flagDebug {
				if whyDisabled != "" {
					debugPrint(fmt.Sprintf("(-) %s is %s", g.Name, whyDisabled))
				} else {
					debugPrint(fmt.Sprintf("(+) %s is enabled", g.Name))
				}
			}
			return whyDisabled == ""
		},
	}

	globalEngine = ruleguard.NewEngine()
	globalEngine.InferBuildContext()

	if err := globalEngine.LoadFromIR(ctx, "rulesdata.go", rulesdata.PrecompiledRules); err != nil {
		return fmt.Errorf("on load ir rules: %s", err)
	}

	if flagRules != "" {
		filenames := strings.Split(flagRules, ",")
		for _, filename := range filenames {
			filename = strings.TrimSpace(filename)
			data, err := ioutil.ReadFile(filename)
			if err != nil {
				return fmt.Errorf("read rules file: %v", err)
			}

			if err = globalEngine.Load(ctx, filename, bytes.NewReader(data)); err != nil {
				return fmt.Errorf("parse rules file: %v", err)
			}
		}
	}

	if flagSkipDirs != "" {
		for _, d := range strings.Split(flagSkipDirs, ",") {
			skipDirsPatterns = append(skipDirsPatterns, regexp.MustCompile(d))
		}
	}

	return nil
}

func main() {
	singlechecker.Main(Analyzer)
}

func runAnalyzer(pass *analysis.Pass) (interface{}, error) {
	if err := prepareEngine(); err != nil {
		return nil, err
	}

	if globalEngine == nil {
		return nil, nil
	}

	ctx := &ruleguard.RunContext{
		DebugPrint: debugPrint,
		Pkg:        pass.Pkg,
		Types:      pass.TypesInfo,
		Sizes:      pass.TypesSizes,
		Fset:       pass.Fset,
		Report: func(data *ruleguard.ReportData) {
			fullMessage := data.Message
			diag := analysis.Diagnostic{
				Pos:     data.Node.Pos(),
				Message: fullMessage,
			}
			if data.Suggestion != nil {
				s := data.Suggestion
				diag.SuggestedFixes = []analysis.SuggestedFix{
					{
						Message: "suggested replacement",
						TextEdits: []analysis.TextEdit{
							{
								Pos:     s.From,
								End:     s.To,
								NewText: s.Replacement,
							},
						},
					},
				}
			}
			pass.Report(diag)
		},
	}

	skippedPaths := make(map[string]struct{})
	for _, f := range pass.Files {
		if _, ok := skippedPaths[pass.Pkg.Path()]; ok || skipDir(pass.Pkg.Path()) {
			if flagDebug {
				debugPrint("dir skipped: " + pass.Pkg.Path())
			}

			skippedPaths[pass.Pkg.Path()] = struct{}{}
			continue
		}

		if err := globalEngine.Run(ctx, f); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func skipDir(dir string) bool {
	for _, pattern := range skipDirsPatterns {
		if pattern.MatchString(dir) {
			return true
		}
	}
	return false
}

func debugPrint(s string) {
	fmt.Println("debug:", s)
}

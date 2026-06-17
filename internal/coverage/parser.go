package coverage

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// FunctionCoverage holds coverage data for a single function within a module.
type FunctionCoverage struct {
	File     string
	Package  string
	Name     string
	Line     int
	Coverage float64
}

// ModuleCoverage holds all coverage data for a single Go module.
type ModuleCoverage struct {
	Error      error
	Dir        string
	ModulePath string
	Functions  []FunctionCoverage
}

var coverLineRegex = regexp.MustCompile(`^([^:]+):(\d+):\s+(\S+)\s+([\d.]+)%`)

func parseCoverOutput(r io.Reader) ([]FunctionCoverage, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var results []FunctionCoverage
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "total:") {
			continue
		}
		matches := coverLineRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		lineNum, err := strconv.Atoi(matches[2])
		if err != nil {
			lineNum = 0
		}
		coverage, err := strconv.ParseFloat(matches[4], 64)
		if err != nil {
			coverage = 0
		}
		results = append(results, FunctionCoverage{
			File:     matches[1],
			Line:     lineNum,
			Name:     matches[3],
			Coverage: coverage,
		})
	}
	return results, nil
}

// func ParseCoverBytes(data []byte) ([]FunctionCoverage, error) {
// 	return parseCoverOutput(bytes.NewReader(data))
// }

type profileEntry struct {
	path    string
	start   int
	covered bool
}

func parseCoverProfile(profilePath, modDir, modPath string) ([]FunctionCoverage, error) {
	file, err := os.Open(profilePath)
	if err != nil {
		return nil, fmt.Errorf("open profile: %w", err)
	}
	defer file.Close()

	entries, err := readProfileEntries(file)
	if err != nil {
		return nil, fmt.Errorf("read profile: %w", err)
	}

	// Group entries by file
	byFile := make(map[string][]profileEntry)
	for _, e := range entries {
		byFile[e.path] = append(byFile[e.path], e)
	}

	var results []FunctionCoverage
	for path, fileEntries := range byFile {
		resolved := resolvePath(modDir, modPath, path)
		funcResults := parseFileProfile(modDir, resolved, fileEntries, modPath)
		results = append(results, funcResults...)
	}
	return results, nil
}

func resolvePath(modDir, modPath, profilePath string) string {
	// Strip module path prefix to get relative file path
	rel := strings.TrimPrefix(profilePath, modPath)
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" {
		return modDir
	}
	return filepath.Join(modDir, rel)
}

type fileFunc struct {
	name      string
	startLine int
	endLine   int
	declLine  int
}

type funcCoverage struct {
	name     string
	declLine int
	total    int
	covered  int
}

func parseFileProfile(modDir, filePath string, entries []profileEntry, modPath string) []FunctionCoverage {
	src, err := os.Open(filePath)
	if err != nil {
		// Coverage file unreadable — skip silently.
		// The function will appear with 0% coverage in the report.
		return nil
	}
	defer src.Close()

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, src, 0)
	if err != nil {
		// Coverage file unreadable — skip silently.
		// The function will appear with 0% coverage in the report.
		return nil
	}

	funcs := extractFileFuncs(fset, node)
	funcMap, orderedFuncs := buildFuncMap(funcs)
	attributeBlocks(fset, node, entries, funcMap)
	results := buildCoverageResults(funcMap, orderedFuncs, filePath, modPath)
	return results
}

func extractFileFuncs(fset *token.FileSet, node *ast.File) []fileFunc {
	var funcs []fileFunc
	ast.Inspect(node, func(n ast.Node) bool {
		fd, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		name := fd.Name.Name
		if fd.Recv != nil {
			recv := extractRecvName(fd.Recv)
			if recv != "" {
				name = recv + "." + name
			}
		}

		startLine := fset.Position(fd.Pos()).Line
		endLine := fset.Position(fd.End()).Line
		declLine := startLine

		if fd.Body != nil {
			declLine = fset.Position(fd.Body.Lbrace).Line
		}

		funcs = append(funcs, fileFunc{
			name:      name,
			startLine: startLine,
			endLine:   endLine,
			declLine:  declLine,
		})

		return true
	})

	return funcs
}

func buildFuncMap(funcs []fileFunc) (map[string]*funcCoverage, []string) {
	funcMap := make(map[string]*funcCoverage)
	var orderedFuncs []string

	for _, ff := range funcs {
		key := ff.name
		if _, exists := funcMap[key]; !exists {
			funcMap[key] = &funcCoverage{name: key, declLine: ff.declLine}
			orderedFuncs = append(orderedFuncs, key)
		}
	}
	return funcMap, orderedFuncs
}

func attributeBlocks(fset *token.FileSet, node *ast.File, entries []profileEntry, funcMap map[string]*funcCoverage) {
	for _, e := range entries {
		fn := findFunctionForBlock(fset, node, e.start)
		if fn != nil {
			key := fn.name
			if fc, ok := funcMap[key]; ok {
				fc.total++
				if e.covered {
					fc.covered++
				}
			}
		}
	}
}

func buildCoverageResults(funcMap map[string]*funcCoverage, orderedFuncs []string, filePath, pkgPath string) []FunctionCoverage {
	var results []FunctionCoverage
	for _, key := range orderedFuncs {
		fc := funcMap[key]
		var coverage float64
		if fc.total > 0 {
			coverage = float64(fc.covered) / float64(fc.total) * 100
		}
		results = append(results, FunctionCoverage{
			File:     filePath,
			Package:  pkgPath,
			Name:     fc.name,
			Line:     fc.declLine,
			Coverage: coverage,
		})
	}
	return results
}

type funcDeclInfo struct {
	name      string
	bodyStart int
	bodyEnd   int
}

func findFunctionForBlock(fset *token.FileSet, node *ast.File, blockLine int) *funcDeclInfo {
	best := findInnermostFunc(fset, node, blockLine)
	if best == nil {
		return nil
	}

	name := resolveReceiverPrefix(node, best.name)
	return &funcDeclInfo{name: name, bodyStart: best.bodyStart, bodyEnd: best.bodyEnd}
}

func findInnermostFunc(fset *token.FileSet, node *ast.File, blockLine int) *funcDeclInfo {
	var best *funcDeclInfo
	ast.Inspect(node, func(n ast.Node) bool {
		fd, ok := n.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			return true
		}

		bodyStart := fset.Position(fd.Body.Lbrace).Line
		bodyEnd := fset.Position(fd.Body.Rbrace).Line
		if (blockLine >= bodyStart) && (blockLine <= bodyEnd) {
			if best == nil {
				best = &funcDeclInfo{}
			}

			if bodyStart > best.bodyStart {
				best.name = fd.Name.Name
				best.bodyStart = bodyStart
				best.bodyEnd = bodyEnd
			}
		}

		return true
	})

	return best
}

func resolveReceiverPrefix(node *ast.File, funcName string) string {
	name := funcName
	for _, fd := range node.Decls {
		funcDecl, ok := fd.(*ast.FuncDecl)
		if !ok || funcDecl.Name.Name != name {
			continue
		}
		if funcDecl.Recv != nil {
			recv := extractRecvName(funcDecl.Recv)
			if recv != "" {
				name = recv + "." + name
				break
			}
		}
	}
	return name
}

func readProfileEntries(r io.Reader) ([]profileEntry, error) {
	var entries []profileEntry
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}
		e, err := parseProfileLine(line)
		if err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, scanner.Err()
}

func parseProfileLine(line string) (profileEntry, error) {
	var entry profileEntry

	colonIdx := strings.Index(line, ":")
	if colonIdx <= 0 {
		return entry, fmt.Errorf("invalid line: %s", line)
	}

	entry.path = line[:colonIdx]
	rest := line[colonIdx+1:]

	startLine, covered, err := parsePositionFields(rest)
	if err != nil {
		return entry, err
	}
	entry.start = startLine
	entry.covered = covered

	return entry, nil
}

func parsePositionFields(rest string) (startLine int, covered bool, err error) {
	spaceIdx := strings.Index(rest, " ")
	if spaceIdx <= 0 {
		return 0, false, fmt.Errorf("invalid line: %s", rest)
	}

	posStr := rest[:spaceIdx]
	parts := strings.Split(posStr, ",")
	if len(parts) != 2 {
		return 0, false, fmt.Errorf("invalid position: %s", posStr)
	}

	startLine, _ = parseCoord(parts[0])

	coveredStr := strings.Fields(rest[spaceIdx+1:])
	if len(coveredStr) < 2 {
		return 0, false, fmt.Errorf("invalid fields: %s", rest)
	}

	coveredInt, err := strconv.Atoi(coveredStr[1])
	if err != nil || coveredInt == 0 {
		return startLine, false, nil
	}
	return startLine, true, nil
}

func parseCoord(s string) (line, col int) {
	before, after, ok := strings.Cut(s, ".")
	if !ok {
		return 0, 0
	}

	line, err := strconv.Atoi(before)
	if err != nil {
		return 0, 0
	}

	col, err = strconv.Atoi(after)
	if err != nil {
		return line, 0
	}

	return line, col
}

func lookupFuncName(modDir, profilePath string) string {
	candidatePath := profilePath
	if !filepath.IsAbs(candidatePath) {
		candidatePath = filepath.Join(modDir, candidatePath)
	}

	src, err := os.Open(candidatePath)
	if err != nil {
		return ""
	}
	defer src.Close()

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, profilePath, src, 0)
	if err != nil {
		return ""
	}

	var funcs []*ast.FuncDecl
	ast.Inspect(node, func(n ast.Node) bool {
		if fd, ok := n.(*ast.FuncDecl); ok {
			funcs = append(funcs, fd)
		}
		return true
	})

	if len(funcs) == 0 {
		return ""
	}

	name := funcs[0].Name.Name
	if funcs[0].Recv != nil {
		recv := extractRecvName(funcs[0].Recv)
		if recv != "" {
			name = recv + "." + name
		}
	}

	return name
}

func extractRecvName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	field := recv.List[0]
	switch t := field.Type.(type) {
	case *ast.StarExpr:
		if sel, ok := t.X.(*ast.Ident); ok {
			return "*" + sel.Name
		}
		if sel, ok := t.X.(*ast.SelectorExpr); ok {
			if x, ok := sel.X.(*ast.Ident); ok {
				return "*" + x.Name
			}
		}
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			return x.Name
		}
	}
	return ""
}

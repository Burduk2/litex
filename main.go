package main

import (
	"fmt"
	"main/engine"
	"os"
	"regexp"
	"strings"
)

func LxError(text string, args ...any) {
	fmt.Printf("litex: "+text+"\n", args...)
	os.Exit(1)
}

func readInputFile(cmd, filePath string) string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		LxError("%s: %s", cmd, err)
	}
	return string(data)
}

func parseSourceArg(cmd, kind string, args []string) (string, []string, bool) {
	if len(args) == 0 {
		LxError("%s: no %s provided", cmd, kind)
	}

	if args[0] == "-f" {
		if len(args) < 2 {
			LxError("%s: -f expects a file path", cmd)
		}
		return readInputFile(cmd, args[1]), args[2:], true
	}

	return args[0], args[1:], false
}

func parsePatternArg(cmd string, args []string) (string, []string) {
	pattern, rest, fromFile := parseSourceArg(cmd, "pattern", args)
	if fromFile {
		return pattern, rest
	}
	return "pattern: " + pattern, rest
}

func printHelp() {
	fmt.Print(`usage:
  lx compile [-f pattern.lx | <pattern>] [--name=value ...]
  lx find [-f pattern.lx | <pattern>] [-f content.txt | <content>] [--name=value ...]
  lx findall [-f pattern.lx | <pattern>] [-f content.txt | <content>] [--name=value ...]
  lx test [-f pattern.lx | <pattern>] [-f content.txt | <content>] [--name=value ...]
  lx replace [-f pattern.lx | <pattern>] <replacement> [-f content.txt | <content>] [--name=value ...]

pattern = inline source by default; inline input implicitly gets a leading pattern:
        use -f to load a full source file as-is
content = inline source by default, or use -f to load from file
`)
}

func parseVarAssignment(raw string) (string, string) {
	name, value, ok := strings.Cut(raw, "=")
	if !ok {
		LxError("invalid variable assignment: " + raw + " (expected name=value)")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		LxError("invalid variable assignment: empty variable name")
	}

	return name, value
}

func parseCliVars(args []string) map[string]string {
	vars := make(map[string]string)

	for i := 0; i < len(args); i++ {
		arg := args[i]

		varExpr, hasPrefix := strings.CutPrefix(arg, "--")
		if hasPrefix {
			name, value := parseVarAssignment(varExpr)
			vars[name] = value
			continue
		}

		LxError("unexpected extra argument: " + arg)
	}

	return vars
}

func main() {
	numArgs := len(os.Args)
	if numArgs < 2 {
		LxError("no command provided\n'lx help' for help")
	}

	options := engine.RunnerOptions{}

	cmd := os.Args[1]
	switch cmd {
	case "help":
		printHelp()
		return
	case "compile":
		if numArgs < 3 {
			LxError("not enough arguments for %s", cmd)
		}
		pattern, rest := parsePatternArg(cmd, os.Args[2:])

		options.Mode = engine.CompileMode
		options.Pattern = pattern
		options.Vars = parseCliVars(rest)
	case "find", "findall", "test":
		pattern, rest := parsePatternArg(cmd, os.Args[2:])
		content, rest, _ := parseSourceArg(cmd, "content", rest)

		var mode engine.RunnerMode
		switch cmd {
		case "find":
			mode = engine.FindMode
		case "findall":
			mode = engine.FindAllMode
		case "test":
			mode = engine.TestMode
		}

		options.Mode = mode
		options.Pattern = pattern
		options.Content = content
		options.Vars = parseCliVars(rest)
	case "replace", "replaceall":
		pattern, rest := parsePatternArg(cmd, os.Args[2:])
		if len(rest) < 2 {
			LxError("not enough arguments for %s", cmd)
		}
		replacement := rest[0]
		content, rest, _ := parseSourceArg(cmd, "content", rest[1:])

		var mode engine.RunnerMode
		switch cmd {
		case "replace":
			mode = engine.ReplaceMode
		case "replaceall":
			mode = engine.ReplaceAllMode
		}

		options.Mode = mode
		options.Pattern = pattern
		options.Content = content
		options.Replacement = replacement
		options.Vars = parseCliVars(rest)
	default:
		LxError("unknown command: %s\n'lx help' for help", cmd)
	}

	result := engine.Run(options)
	fmt.Println(result)
}

func FindCaptures(re *regexp.Regexp, str string) (map[string][]string, bool) {
	all := re.FindAllStringSubmatch(str, -1)
	if len(all) == 0 {
		return nil, false
	}
	names := re.SubexpNames()
	if len(names) <= 1 {
		return nil, false
	}

	captures := make(map[string][]string, len(names)-1)
	for _, match := range all {
		for i := 1; i < len(match); i++ { // skip index 0 (whole match)
			name := names[i]
			captures[name] = append(captures[name], match[i])
		}
	}

	return captures, true
}

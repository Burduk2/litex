package main

import (
	"fmt"
	"main/engine"
	"os"
	"strings"
)

func LxError(text string) {
	fmt.Println("litex: " + text)
	os.Exit(1)
}

func isCliFilePath(str string) bool {
	if str == "" {
		return false
	}
	return str[0] == '@'
}

func handlePatternContentArgs(cmd, pattern, content string) (string, string) {
	if pattern == "" {
		LxError(cmd + ": no pattern provided")
	}

	results := []string{pattern, content}
	for i := 0; i < len(results); i++ {
		arg := results[i]
		if !isCliFilePath(arg) {
			continue
		}
		filePath := arg[1:]
		data, err := os.ReadFile(filePath)
		if err != nil {
			LxError(fmt.Sprintf("%s: %s", cmd, err))
		}
		results[i] = string(data)
	}

	return results[0], results[1]
}

func printHelp() {
	fmt.Print(`usage:
	lx compile <pattern> [--name=value ...]
  lx find <pattern> <content> [--name=value ...]
  lx findall <pattern> <content> [--name=value ...]
  lx test <pattern> <content> [--name=value ...]
  lx replace <pattern> <replacement> <content> [--name=value ...]

pattern = inline source or @file.lx
content = inline source or @file

custom vars:
  --name=value       shorthand form
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
			LxError(fmt.Sprintf("not enough arguments for %s", cmd))
		}
		pattern, _ := handlePatternContentArgs(cmd, os.Args[2], "")

		options.Mode = engine.CompileMode
		options.Pattern = pattern
		options.Vars = parseCliVars(os.Args[3:])
	case "find", "findall", "test":
		if numArgs < 4 {
			LxError(fmt.Sprintf("not enough arguments for %s", cmd))
		}
		pattern, content := handlePatternContentArgs(cmd, os.Args[2], os.Args[3])

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
		options.Vars = parseCliVars(os.Args[4:])
	case "replace", "replaceall":
		if numArgs < 5 {
			LxError(fmt.Sprintf("not enough arguments for %s", cmd))
		}
		pattern, content := handlePatternContentArgs(cmd, os.Args[2], os.Args[4])

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
		options.Replacement = os.Args[3]
		options.Vars = parseCliVars(os.Args[5:])
	default:
		LxError(fmt.Sprintf("unknown command: %s\n'lx help' for help", cmd))
	}

	result := engine.Run(options)
	fmt.Println(result)
}

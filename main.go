package main

import (
	"fmt"
	"main/engine"
	"os"
)

func LxError(text string) {
	fmt.Println("lirex: " + text)
	os.Exit(1)
}

func isCliFilePath(str string) bool {
	if str == "" {
		return false
	}
	return str[0] == '@'
}

func handleFindTestArgs(cmd, pattern, content string) (string, string) {
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

func main() {
	numArgs := len(os.Args)
	if numArgs < 2 {
		LxError("no command provided\n'lx help' for help")
	}

	pattern, content, replacement := "", "", ""
	vars := make(map[string]string)

	cmd := os.Args[1]
	switch cmd {
	case "help":
		fmt.Println("hell naw man")
	case "find", "findall", "test":
		if numArgs < 4 {
			LxError(fmt.Sprintf("not enough arguments for %s", cmd))
		}
		pattern, content = handleFindTestArgs(cmd, os.Args[2], os.Args[3])
	case "replace", "replaceall":
		fmt.Println("replace")
	default:
		LxError(fmt.Sprintf("unknown command: %s\n'lx help' for help", cmd))
	}

	result := engine.Run(cmd, pattern, content, replacement, vars)
	fmt.Println(result)
}

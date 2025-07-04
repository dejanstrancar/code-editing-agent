package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"code-editing-agent/agent"
	"code-editing-agent/tools"

	"github.com/anthropics/anthropic-sdk-go"
)

func main() {
	client := anthropic.NewClient()
	scanner := bufio.NewScanner(os.Stdin)
	getUserMessage := func() (string, bool) {
		if !scanner.Scan() {
			return "", false
		}
		return scanner.Text(), true
	}

	toolDefinitions := []tools.ToolDefinition{tools.ReadFileDefinition, tools.ListFilesDefinition, tools.EditFileDefinition}
	agent := agent.NewAgent(&client, getUserMessage, toolDefinitions)
	err := agent.Run(context.TODO())
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}

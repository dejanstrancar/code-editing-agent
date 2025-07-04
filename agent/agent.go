package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"code-editing-agent/tools"

	"github.com/anthropics/anthropic-sdk-go"
)

type Agent struct {
	client         *anthropic.Client
	getUserMessage func() (string, bool)
	tools          map[string]tools.ToolDefinition
}

func NewAgent(client *anthropic.Client, getUserMessage func() (string, bool), toolDefinitions []tools.ToolDefinition) *Agent {
	newAgent := &Agent{
		client:         client,
		getUserMessage: getUserMessage,
		tools:          make(map[string]tools.ToolDefinition),
	}
	for _, tool := range toolDefinitions {
		newAgent.tools[tool.Name] = tool
	}
	return newAgent
}

func (a *Agent) Run(ctx context.Context) error {
	conversation := []anthropic.MessageParam{}

	fmt.Println("Chat with Claude (use 'ctrl-c' to quit)")

	readUserInput := true
	for {
		if readUserInput {
			fmt.Print("\u001b[94mYou\u001b[0m: ")
			userInput, ok := a.getUserMessage()
			if !ok {
				break
			}

			userMessage := anthropic.NewUserMessage(anthropic.NewTextBlock(userInput))
			conversation = append(conversation, userMessage)
		}

		message, err := a.runInference(ctx, conversation)
		if err != nil {
			return err
		}
		conversation = append(conversation, message.ToParam())

		toolResults := []anthropic.ContentBlockParamUnion{}
		for _, content := range message.Content {
			switch content.Type {
			case "text":
				fmt.Printf("\u001b[93mClaude\u001b[0m: %s\n", content.Text)
			case "tool_use":
				result := a.executeTool(content.ID, content.Name, content.Input)
				toolResults = append(toolResults, result)
			}
		}
		if len(toolResults) == 0 {
			readUserInput = true
			continue
		}
		readUserInput = false
		conversation = append(conversation, anthropic.NewUserMessage(toolResults...))
	}

	return nil
}

func (a *Agent) runInference(ctx context.Context, conversation []anthropic.MessageParam) (*anthropic.Message, error) {
	resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude_3_Opus_20240229,
		Messages:  conversation,
		MaxTokens: 4096,
		Tools:     a.getTools(),
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (a *Agent) getTools() []anthropic.ToolUnionParam {
	var toolParams []anthropic.ToolParam
	for _, tool := range a.tools {
		toolParams = append(toolParams, anthropic.ToolParam{
			Name:        tool.Name,
			Description: anthropic.String(tool.Description),
			InputSchema: tool.InputSchema,
		})
	}
	var toolUnionParams []anthropic.ToolUnionParam
	for i := range toolParams {
		toolUnionParams = append(toolUnionParams, anthropic.ToolUnionParam{OfTool: &toolParams[i]})
	}
	return toolUnionParams
}

func (a *Agent) executeTool(id, name string, input json.RawMessage) anthropic.ContentBlockParamUnion {
	tool, ok := a.tools[name]
	if !ok {
		return anthropic.NewToolResultBlock(id, "tool not found", true)
	}

	fmt.Printf("\u001b[92mtool\u001b[0m: %s(%s)\n", name, input)
	response, err := tool.Function(input)
	if err != nil {
		return anthropic.NewToolResultBlock(id, err.Error(), true)
	}
	return anthropic.NewToolResultBlock(id, response, false)
}

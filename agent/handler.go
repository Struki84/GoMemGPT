package agent

import (
	"context"
	"log"

	"github.com/Struki84/GoMemGPT/memory"
	"github.com/tmc/langchaingo/llms"
)

type Handler interface {
	Call(ctx context.Context, memory map[string]any, userMsg llms.MessageContent) (llms.MessageContent, error)
	GetMemory() memory.Manager
}

func Run(ctx context.Context, agent Handler, userPrompt string) (string, error) {
	memory := agent.GetMemory().LoadMemory(ctx)

	userMsg := llms.TextParts(llms.ChatMessageTypeHuman, userPrompt)

	agentMsg, err := agent.Call(ctx, memory, userMsg)
	if err != nil {
		log.Printf("Error calling agent: %v", err)
		return "", err
	}

	err = agent.GetMemory().SaveMemory(ctx, userMsg, agentMsg)
	if err != nil {
		log.Printf("Error saving memory: %v", err)
		return "", err
	}

	return agentMsg.Parts[0].(llms.TextContent).String(), nil
}

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Struki84/GoMemGPT/agent"
	"github.com/Struki84/GoMemGPT/memory/storage"
	"github.com/tmc/langchaingo/llms/openai"
)

var chat agent.Handler

func init() {
	memoryStorage := storage.NewSqliteStorage()
	llm, err := openai.New(
		openai.WithModel("gpt-4o"),
	)

	if err != nil {
		log.Printf("Error initializing LLM: %v", err)
	}

	chat = agent.NewChatAgent(llm, memoryStorage)
}

func main() {
	ctx := context.Background()

	response, err := agent.Run(ctx, chat, "hello")
	if err != nil {
		log.Printf("Error running agent: %v", err)
		return
	}

	fmt.Printf("Agent Response: %s\n", response)
}

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Struki84/GoMemGPT/memory"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

func Agent(input string) {
	ctx := context.Background()

	llm, err := openai.New(
		openai.WithModel("gpt-4o"),
	)
	if err != nil {
		log.Printf("Error initializing LLM: %v", err)
	}

	memory := memory.NewMemoryManager(llm, nil)
	msgs := make([]llms.MessageContent, 0)

	userMsg := llms.TextParts(llms.ChatMessageTypeHuman, input)

	err = memory.Update(userMsg)
	if err != nil {
		log.Printf("Error updating memory: %v", err)
	}

	stream := func(ctx context.Context, chunk []byte) error {
		fmt.Println(string(chunk))
		return nil
	}

	msgs = append(msgs, memory.RecallContext())
	msgs = append(msgs, userMsg)

	response, err := llm.GenerateContent(ctx, msgs, llms.WithStreamingFunc(stream))
	if err != nil {
		log.Printf("Error generating response: %v", err)
	}

	responseMsg := llms.TextParts(llms.ChatMessageTypeAI, response.Choices[0].Content)

	err = memory.Update(responseMsg)
	if err != nil {
		log.Printf("Error updating memory: %v", err)
	}
}

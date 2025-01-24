package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Struki84/GoMemGPT/memory"
	"github.com/Struki84/GoMemGPT/memory/storage"
	"github.com/tmc/langchaingo/llms/openai"
)

var chatAgent memory.Agent

func init() {
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	memoryStorage := storage.NewSqliteStorage()
	llm, err := openai.New(
		openai.WithModel("gpt-4o"),
	)

	if err != nil {
		log.Printf("Error initializing LLM: %v", err)
	}

	chatAgent = memory.NewAgent(ctx, llm, memoryStorage)

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Type a message (or 'exit' to quit): ")

	for {
		fmt.Printf("Input: > ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())

		if input == "exit" {
			fmt.Println("Goodbye!")
			break
		}

		// Create a channel so we can wait for the response.
		responseChan := make(chan string)

		// Call the agent, but in the callback, we send the response into responseChan
		chatAgent.Call(input, func(msg string) {
			responseChan <- msg
		})

		// BLOCK here until we get a message from responseChan
		outMsg := <-responseChan
		fmt.Println("Output: >", outMsg)
	}
}

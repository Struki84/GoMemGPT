package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/Struki84/GoMemGPT/memory"
	"github.com/tmc/langchaingo/llms"
)

type Agent struct {
	processor *memory.LLMProcessor
	ready     *sync.WaitGroup
}

func NewAgent(ctx context.Context, llm llms.Model, storage memory.MemoryStorage) Agent {

	wg := &sync.WaitGroup{}
	wg.Add(1)

	mainContext := memory.NewMemoryContext(storage)
	proc := memory.NewLLMProcessor(llm, mainContext)

	go proc.Run(ctx, wg)

	return Agent{
		processor: proc,
		ready:     wg,
	}
}

func (agent *Agent) Call(input string, output func(string)) {
	agent.ready.Wait()

	agent.processor.Output(func(msg llms.MessageContent) {
		output(msg.Parts[0].(llms.TextContent).String())
	})

	// log.Printf("Calling agent with input: %s", input)
	fmt.Println("<<<< Internal Messages >>>>")

	userMsg := llms.TextParts(llms.ChatMessageTypeHuman, input)

	err := agent.processor.System.AppendMessage(userMsg)
	if err != nil {
		log.Printf("Error appending user message: %v", err)
	}

	agent.processor.Input(userMsg)
	agent.processor.CheckMemoryPressure()
}

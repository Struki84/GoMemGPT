package memory

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/tmc/langchaingo/llms"
)

type Agent struct {
	llm       llms.Model
	processor *LLMProcessor
	memory    *MemoryContext
	system    *SystemMonitor
	ready     *sync.WaitGroup
}

func NewAgent(ctx context.Context, llm llms.Model, storage MemoryStorage) Agent {

	wg := &sync.WaitGroup{}
	wg.Add(1)

	memory := NewMemoryContext(storage)
	proc := NewLLMProcessor(llm, memory)

	go proc.Run(ctx, wg)

	return Agent{
		llm:       llm,
		processor: proc,
		memory:    memory,
		ready:     wg,
	}
}

func (agent *Agent) Call(input string, output func(string)) {
	agent.ready.Wait()

	agent.processor.Output(func(msg llms.MessageContent) {
		output(msg.Parts[0].(llms.TextContent).String())
	})

	log.Printf("Calling agent with input: %s", input)
	sysPrompt, err := agent.system.Instruction("assistant", map[string]any{
		"time":           time.Now().Format("January 02, 2006"),
		"workingContext": agent.memory.WorkingContext,
	})

	if err != nil {
		log.Printf("Error formatting prompt: %v", err)
		return
	}

	// log.Printf("System prompt: %s", sysPrompt)

	sysMsg := llms.TextParts(llms.ChatMessageTypeSystem, sysPrompt)
	userMsg := llms.TextParts(llms.ChatMessageTypeHuman, input)

	agent.memory.Messages = append(agent.memory.Messages, sysMsg)
	agent.memory.Messages = append(agent.memory.Messages, userMsg)

	agent.processor.Input(userMsg)

	agent.system.InspectMemoryPressure(agent)
}

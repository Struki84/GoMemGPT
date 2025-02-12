package memory

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/Struki84/GoMemGPT/logger"
	"github.com/tmc/langchaingo/llms"
)

type step struct {
	msg llms.MessageContent
}

type LLMProcessor struct {
	llm      llms.Model
	System   *SystemMonitor
	mainProc chan llms.MessageContent
	executor Executor
	output   func(llms.MessageContent)
}

func NewLLMProcessor(llm llms.Model, mainContext *MemoryContext) *LLMProcessor {

	exec := NewExecutor(mainContext)

	// load chat history.
	// tmp solution since I don't like it this way
	exec.operator.Load()

	return &LLMProcessor{
		llm:      llm,
		System:   NewSystemMonitor(mainContext),
		executor: NewExecutor(mainContext),
		mainProc: make(chan llms.MessageContent, 100),
	}
}

func (processor *LLMProcessor) Input(msg llms.MessageContent) {
	processor.mainProc <- msg
}

func (processor *LLMProcessor) Output(fn func(llms.MessageContent)) {
	processor.output = fn
}

func (processor *LLMProcessor) Run(ctx context.Context, wg *sync.WaitGroup) {
	wg.Done()

	for {
		select {
		case msg, ok := <-processor.mainProc:
			if !ok {
				log.Printf("Processor mainProc channel closed")
				return
			}

			processor.handleMessage(ctx, msg)
		case <-ctx.Done():
			log.Printf("Processor mainProc loop done")
			return
		}
	}
}

func (processor *LLMProcessor) handleMessage(ctx context.Context, msg llms.MessageContent) {
	logger.LogLastMessage(processor.System.mainContext.Messages)

	switch msg.Role {
	case llms.ChatMessageTypeHuman:
		processor.callLLM(ctx)
	case llms.ChatMessageTypeSystem:
		processor.System.AppendMessage(msg)
		processor.callLLM(ctx)
	case llms.ChatMessageTypeTool:
		output := false

		for _, part := range msg.Parts {
			if toolResponse, ok := part.(llms.ToolCallResponse); ok {
				if toolResponse.Name == "InternalOutput" {
					output = true
					newMsg := llms.TextParts(llms.ChatMessageTypeAI, toolResponse.Content)
					processor.System.AppendMessage(newMsg)
					processor.CheckMemoryPressure()
				}

				if toolResponse.Name == "ExternalOutput" {
					output = true
					newMsg := llms.TextParts(llms.ChatMessageTypeAI, toolResponse.Content)
					processor.System.AppendMessage(newMsg)
					processor.CheckMemoryPressure()
					processor.mainProc <- newMsg
				}
			}
		}

		if !output {
			processor.callLLM(ctx)
		}
	case llms.ChatMessageTypeAI:
		tool := false

		for _, part := range msg.Parts {
			if toolCall, ok := part.(llms.ToolCall); ok {
				tool = true

				executionResult, err := processor.executor.Run(toolCall)
				if err != nil {
					executionResult = fmt.Sprintf("Error running function: %v", err)
				}

				newMsg := llms.MessageContent{
					Role: llms.ChatMessageTypeTool,
					Parts: []llms.ContentPart{
						llms.ToolCallResponse{
							ToolCallID: toolCall.ID,
							Name:       toolCall.FunctionCall.Name,
							Content:    executionResult,
						},
					},
				}

				processor.System.AppendMessage(newMsg)
				processor.CheckMemoryPressure()

				processor.mainProc <- newMsg
			}
		}

		if !tool && processor.output != nil {
			processor.output(msg)
		}

	}
}

func (processor *LLMProcessor) callLLM(ctx context.Context) {
	log.Println(processor.System.mainContext.Messages)
	response, err := processor.llm.GenerateContent(ctx, processor.System.mainContext.Messages,
		llms.WithTools(processor.executor.functions),
	)

	if err != nil {
		log.Printf("Error generating response: %v", err)
		return
	}

	newMsg := llms.TextParts(llms.ChatMessageTypeAI, response.Choices[0].Content)

	if len(response.Choices[0].ToolCalls) > 0 {
		newMsg = llms.TextParts(llms.ChatMessageTypeAI, "preforming function calls")
		for _, toolCall := range response.Choices[0].ToolCalls {
			newMsg.Parts = append(newMsg.Parts, toolCall)
		}
	}

	processor.System.AppendMessage(newMsg)
	processor.mainProc <- newMsg
}

func (processor *LLMProcessor) CheckMemoryPressure() {
	systemWarrnings := processor.System.InspectMemoryPressure()

	for _, systemWarrning := range systemWarrnings {
		processor.Input(systemWarrning)
	}
}

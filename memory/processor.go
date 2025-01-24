package memory

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/tmc/langchaingo/llms"
)

type LLMProcessor struct {
	llm         llms.Model
	mainContext *MemoryContext
	mainProc    chan llms.MessageContent
	executor    Executor
	output      func(llms.MessageContent)
}

func NewLLMProcessor(llm llms.Model, mainContext *MemoryContext) *LLMProcessor {

	return &LLMProcessor{
		llm:         llm,
		mainContext: mainContext,
		executor:    NewExecutor(mainContext),
		mainProc:    make(chan llms.MessageContent, 100),
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

	switch msg.Role {
	case llms.ChatMessageTypeSystem, llms.ChatMessageTypeHuman:
		processor.callLLM(ctx)
	case llms.ChatMessageTypeFunction:
		output := false

		for _, part := range msg.Parts {
			if toolCall, ok := part.(llms.ToolCall); ok {
				if toolCall.FunctionCall.Name == "InternalOutput" {
					output = true
					newMsg := llms.TextParts(llms.ChatMessageTypeAI, msg.Parts[0].(llms.TextContent).String())
					processor.mainContext.Messages = append(processor.mainContext.Messages, newMsg)
				}

				if toolCall.FunctionCall.Name == "ExternalOutput" {
					output = true
					newMsg := llms.TextParts(llms.ChatMessageTypeAI, msg.Parts[0].(llms.TextContent).String())
					processor.mainContext.Messages = append(processor.mainContext.Messages, newMsg)
					processor.output(newMsg)
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
					newMsg := llms.TextParts(llms.ChatMessageTypeFunction, fmt.Sprintf("Error running function: %v", err))
					newMsg.Parts = append(newMsg.Parts, toolCall)

					processor.mainContext.Messages = append(processor.mainContext.Messages, newMsg)
					processor.mainProc <- newMsg
				} else {
					newMsg := llms.TextParts(llms.ChatMessageTypeFunction, executionResult)
					newMsg.Parts = append(newMsg.Parts, toolCall)

					processor.mainContext.Messages = append(processor.mainContext.Messages, newMsg)
					processor.mainProc <- newMsg
				}
			}
		}

		if !tool && processor.output != nil {
			processor.output(msg)
		}
	}
}

func (processor *LLMProcessor) callLLM(ctx context.Context) {
	response, _ := processor.llm.GenerateContent(ctx, processor.mainContext.Messages,
		llms.WithTools(processor.executor.functions),
	)

	newMsg := llms.TextParts(llms.ChatMessageTypeAI, response.Choices[0].Content)

	for _, toolCall := range response.Choices[0].ToolCalls {
		newMsg.Parts = append(newMsg.Parts, toolCall)
	}

	processor.mainContext.Messages = append(processor.mainContext.Messages, newMsg)
	processor.mainProc <- newMsg
}

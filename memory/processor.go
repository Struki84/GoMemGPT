package memory

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
)

type LLMProcessor struct {
	llm         llms.Model
	mainContext *MemoryContext
	mainProc    chan llms.MessageContent
	resultProc  chan llms.MessageContent
	executor    Executor
	output      func(llms.MessageContent)
}

func NewLLMProcessor(llm llms.Model, mainContext *MemoryContext) *LLMProcessor {

	return &LLMProcessor{
		llm:         llm,
		mainContext: mainContext,
		executor:    NewExecutor(mainContext),
		mainProc:    make(chan llms.MessageContent, 100),
		resultProc:  make(chan llms.MessageContent, 100),
	}
}

func (processor *LLMProcessor) Input(msg llms.MessageContent) {
	processor.mainProc <- msg
}

func (processor *LLMProcessor) Output(fn func(llms.MessageContent)) {
	processor.output = fn
}

func (processor *LLMProcessor) Run(ctx context.Context) {
	go func() {
		for msg := range processor.resultProc {
			processor.mainProc <- msg
		}
	}()

	for msg := range processor.mainProc {
		switch msg.Role {
		case llms.ChatMessageTypeSystem, llms.ChatMessageTypeFunction:
			response, _ := processor.llm.GenerateContent(
				ctx,
				processor.mainContext.Messages,
				llms.WithTools(processor.executor.functions),
			)

			newMsg := llms.TextParts(llms.ChatMessageTypeAI, response.Choices[0].Content)

			if len(response.Choices[0].ToolCalls) > 0 {
				for _, toolCall := range response.Choices[0].ToolCalls {
					newMsg.Parts = append(newMsg.Parts, toolCall)
				}
			}

			processor.mainContext.Messages = append(processor.mainContext.Messages, newMsg)
			processor.resultProc <- newMsg
		case llms.ChatMessageTypeAI:
			if len(msg.Parts) > 0 {
				for _, part := range msg.Parts {

					if toolCall, ok := part.(llms.ToolCall); ok {
						executionResult, err := processor.executor.Run(toolCall)
						if err != nil {
							newMsg := llms.TextParts(llms.ChatMessageTypeFunction, fmt.Sprintf("Error running function: %v", err))

							processor.mainContext.Messages = append(processor.mainContext.Messages, newMsg)
							processor.resultProc <- newMsg
						}

						newMsg := llms.TextParts(llms.ChatMessageTypeFunction, executionResult)
						processor.mainContext.Messages = append(processor.mainContext.Messages, newMsg)
						processor.resultProc <- newMsg
					}

				}
			} else {
				processor.output(msg)
			}
		}
	}
}

package memory

import (
	"context"
	"log"

	"github.com/pkoukk/tiktoken-go"
	"github.com/tmc/langchaingo/llms"
)

var (
	defaultContextSize       int     = 1024
	defaultSysMsgSize        int     = 4086
	defaultHistoricalCtxSize float32 = 0.25
	defaultWorkingCtxSize    float32 = 0.25
	defaulHistorySize        float32 = 0.50
)

type Manager interface {
	SaveMemory(ctx context.Context, input, output llms.MessageContent) error

	// needs .historicalContext and .currentContext
	// needs .conversationHistory buffer string
	LoadMemory(ctx context.Context) map[string]any
}

// this is my queue manager - The queue manager manages messages in recall storage
// and the FIFO queue.
type MemoryManager struct {
	processor      *LLMProcessor
	mainContext    *MemoryContext
	tokenEncoder   *tiktoken.Tiktoken
	maxContextSize int
	workingCtxSize int
	historySize    int
}

func NewMemoryManager(llm llms.Model, storage MemoryStorage) *MemoryManager {
	mainContext := NewMemoryContext(storage)

	encoder, err := tiktoken.EncodingForModel("gpt-4o")
	if err != nil {
		log.Printf("Error initializing encoding: %v", err)
	}

	primerSize := encoder.Encode(mainContext.SystemInstructions["assistant"], nil, nil)

	maxContextSize := defaultContextSize - len(primerSize)

	proc := NewLLMProcessor(llm, mainContext)

	go proc.Run(context.Background())

	return &MemoryManager{
		processor:      proc,
		mainContext:    mainContext,
		maxContextSize: maxContextSize,
		workingCtxSize: int(float32(maxContextSize) * defaultWorkingCtxSize),
		historySize:    int(float32(maxContextSize) * defaulHistorySize),
		tokenEncoder:   encoder,
	}
}

func (manager *MemoryManager) LoadMemory(ctx context.Context) map[string]any {
	err := manager.mainContext.Load()
	if err != nil {
		log.Printf("Error loading memory: %v", err)
		return map[string]any{}
	}

	chatHistory, err := llms.GetBufferString(manager.mainContext.ChatHistory, "User: ", "AI: ")
	if err != nil {
		log.Printf("Error loading memory: %v", err)
		return map[string]any{}
	}

	return map[string]any{
		"currentContext":      manager.mainContext.WorkingContext,
		"historicalContext":   manager.mainContext.HistoricalContext,
		"conversationHistory": chatHistory,
	}
}

func (manager *MemoryManager) SaveMemory(ctx context.Context, input, output llms.MessageContent) error {
	manager.mainContext.Messages = append(manager.mainContext.Messages, input)
	manager.mainContext.Messages = append(manager.mainContext.Messages, output)

	manager.appendChatMsg(input)
	manager.appendChatMsg(output)

	manager.mainContext.Save()

	// Check memory pressure
	chatHistory, err := llms.GetBufferString(manager.mainContext.ChatHistory, "User: ", "AI: ")
	if err != nil {
		log.Printf("Error loading chat history: %v", err)
		return err
	}

	chatHistorySize := manager.tokenEncoder.Encode(chatHistory, nil, nil)
	if len(chatHistorySize) >= manager.historySize {
		// flush chat history messages
		prompt, err := manager.mainContext.SystemInstruction("memoryPressure:ChatHistory", map[string]any{
			"chatHistory":     chatHistory,
			"chatHistorySize": len(chatHistorySize),
		})
		if err != nil {
			log.Printf("Error formatting prompt: %v", err)
			return err
		}

		sysMsg := llms.TextParts(llms.ChatMessageTypeSystem, prompt)
		manager.processor.Input(sysMsg)
	}

	workingContextSize := manager.tokenEncoder.Encode(manager.mainContext.WorkingContext, nil, nil)
	if len(workingContextSize) >= manager.workingCtxSize {
		// flush working context
		prompt, err := manager.mainContext.SystemInstruction("memoryPressure:WorkingContext", map[string]any{
			"workingContext":     manager.mainContext.WorkingContext,
			"workingContextSize": len(workingContextSize),
		})

		if err != nil {
			log.Printf("Error formatting prompt: %v", err)
			return err
		}

		sysMsg := llms.TextParts(llms.ChatMessageTypeSystem, prompt)
		manager.processor.Input(sysMsg)
	}

	return nil
}

func (manager *MemoryManager) appendChatMsg(msg llms.MessageContent) {
	if msg.Role == llms.ChatMessageTypeHuman {
		chatMsg := llms.HumanChatMessage{
			Content: msg.Parts[0].(llms.TextContent).String(),
		}

		manager.mainContext.ChatHistory = append(manager.mainContext.ChatHistory, chatMsg)
	}

	if msg.Role == llms.ChatMessageTypeAI {
		chatMsg := llms.AIChatMessage{
			Content: msg.Parts[0].(llms.TextContent).String(),
		}

		manager.mainContext.ChatHistory = append(manager.mainContext.ChatHistory, chatMsg)
	}
}

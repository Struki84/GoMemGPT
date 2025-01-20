package memory

import (
	"log"
	"time"

	"github.com/pkoukk/tiktoken-go"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/prompts"
)

var (
	DefaultContextSize    int     = 1024
	DefaultWorkingCtxSize float32 = 0.25
	DefaulHistorySize     float32 = 0.75
)

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

	maxContextSize := DefaultContextSize - len(primerSize)

	proc := NewLLMProcessor(llm, &MemoryContext{})

	go proc.Run()

	return &MemoryManager{
		processor:      proc,
		mainContext:    mainContext,
		maxContextSize: maxContextSize,
		workingCtxSize: int(float32(maxContextSize) * DefaultWorkingCtxSize),
		historySize:    int(float32(maxContextSize) * DefaulHistorySize),
		tokenEncoder:   encoder,
	}
}

func (cm *MemoryManager) RecallContext() llms.MessageContent {
	// generate system instruction representing the agent primer
	// containing conversation history, current context, and self referencing
	// history

	chatHistory, err := llms.GetBufferString(cm.mainContext.ChatHistory, "User: ", "AI: ")
	if err != nil {
		log.Printf("Error loading chat history: %v", err)
		return llms.TextParts(llms.ChatMessageTypeSystem, "Error loading chat history")
	}

	promptTemplate := prompts.PromptTemplate{
		Template:       cm.mainContext.SystemInstructions["assistant"],
		TemplateFormat: prompts.TemplateFormatGoTemplate,
		InputVariables: []string{},
		PartialVariables: map[string]any{
			"currentContext":      cm.mainContext.WorkingContext,
			"historicalContext":   cm.mainContext.HistoricalContext,
			"conversationHistory": chatHistory,
			"time":                time.Now().Format("January 02, 2006"),
		},
	}

	prompt, err := promptTemplate.Format(map[string]any{})
	if err != nil {
		log.Printf("Error formatting prompt: %v", err)
		return llms.TextParts(llms.ChatMessageTypeSystem, "Error formatting prompt")
	}

	return llms.TextParts(llms.ChatMessageTypeSystem, prompt)
}

func (manager *MemoryManager) Input(msg llms.MessageContent) {
	manager.processor.Input(msg)
}

func (manager *MemoryManager) Output(fn func(llms.MessageContent)) {
	manager.processor.Output(fn)
}

func (manager *MemoryManager) Update(userMsg, AIMsg llms.MessageContent) error {
	manager.mainContext.Messages = append(manager.mainContext.Messages, userMsg)
	manager.mainContext.Messages = append(manager.mainContext.Messages, AIMsg)

	manager.SaveChatHistory(userMsg)
	manager.SaveChatHistory(AIMsg)

	// Check memory pressure
	chatHistory, err := llms.GetBufferString(manager.mainContext.ChatHistory, "User: ", "AI: ")
	if err != nil {
		log.Printf("Error loading chat history: %v", err)
		return err
	}

	chatHistorySize := manager.tokenEncoder.Encode(chatHistory, nil, nil)
	if len(chatHistorySize) >= manager.historySize {
		// flush chat history messages
		template := prompts.PromptTemplate{
			Template:       manager.mainContext.SystemInstructions["memoryPressure:ChatHistory"],
			TemplateFormat: prompts.TemplateFormatGoTemplate,
			InputVariables: []string{},
			PartialVariables: map[string]any{
				"chatHistory": chatHistory,
			},
		}

		msgTmplt, err := template.Format(map[string]any{})
		if err != nil {
			log.Printf("Error formatting prompt: %v", err)
			return err
		}

		sysMsg := llms.TextParts(llms.ChatMessageTypeSystem, msgTmplt)

		manager.processor.Input(sysMsg)
	}

	workingContextSize := manager.tokenEncoder.Encode(manager.mainContext.WorkingContext, nil, nil)
	if len(workingContextSize) >= manager.workingCtxSize {
		// flush working context
		template := prompts.PromptTemplate{
			Template:       manager.mainContext.SystemInstructions["memoryPressure:WorkingContext"],
			TemplateFormat: prompts.TemplateFormatGoTemplate,
			InputVariables: []string{},
			PartialVariables: map[string]any{
				"workingContext": manager.mainContext.WorkingContext,
			},
		}

		msgTmplt, err := template.Format(map[string]any{})
		if err != nil {
			log.Printf("Error formatting prompt: %v", err)
			return err
		}

		sysMsg := llms.TextParts(llms.ChatMessageTypeSystem, msgTmplt)

		manager.processor.Input(sysMsg)
	}

	return nil
}

func (manager *MemoryManager) SaveChatHistory(msg llms.MessageContent) {
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

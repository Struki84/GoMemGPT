package memory

import (
	"encoding/json"

	"github.com/tmc/langchaingo/llms"
)

var (
	PrimerTemplate = `
	{{.time}}
	
	You are a helpful assistant. 

	Your brief history is as follows:
	{{.historicalContext}}

	Your current context is as follows:
	{{.currentContext}}

	Your conversation history with the user is as follows:
	{{.conversationHistory}}
	`

	MemoryPressureWorkingContext = `
	{{.time}}
	
	Memory pressure warning: WorkingContext
	
	Working context:
	{{.workingContext}}

	Working context size: {{.workingContextSize}}
	`

	MemoryPressureHistoricalContext = `
	{{.time}}
	
	Memory pressure warning: HistoricalContext
	
	Historical context:
	{{.historicalContext}}

	Working context size: {{.historicalContextSize}}
	`

	MemoryPressureChatHistory = `
	{{.time}}
	
	Memory pressure warning: ChatHistory
	
	Historical context:
	{{.historicalContext}}

	Working context size: {{.historicalContextSize}}
	`

	MemoryPressureMessages = `
	{{.time}}
	
	Memory pressure warning: Messages
	
	Messages in context:
	{{.messages}}

	Messages size: {{.messagesSize}}
	`
)

type MemoryStorage interface {
	LoadMessages() ([]llms.MessageContent, error)
	SaveMessages(messages []llms.MessageContent) error

	LoadChatHistory() ([]llms.ChatMessage, error)
	SaveChatHistory(chatHistory []llms.ChatMessage) error

	LoadWorkingContext() (string, error)
	SaveWorkingContext(workingContext string) error

	LoadHistoricalContext() (string, error)
	SaveHistoricalContext(historicalContext string) error

	RecallMessages() ([]llms.MessageContent, error)
	ArchiveMessages(messages []llms.MessageContent) error

	RecallChatHistory() ([]llms.ChatMessage, error)
	ArchiveChatHistory(chatHistory []llms.ChatMessage) error

	RecallWorkingContext() (string, error)
	ArchiveWorkingContext(workingContext string) error

	RecallHistoricalContext() (string, error)
	ArchiveHistoricalContext(historicalContext string) error

	SearchMessagesArchives(query string) ([]llms.MessageContent, error)
	SearchChatHistoryArchives(query string) ([]llms.ChatMessage, error)
}

// Main context
type MemoryContext struct {
	// FIFO Message Queue, stores a rolling history of messages,
	// including  messages between the agent and user, as well as system
	// messages (e.g. memory warnings) and function call inputs
	// and outputs. The first indehx in the FIFO queue stores a system
	// message containing a recursive summary of messages that have
	// been evicted from the queue.
	Messages []llms.MessageContent

	// Chat history contains the conversation history between the agent and user
	ChatHistory []llms.ChatMessage

	// Current working context
	// Working context is a fixed-size read/write block of unstructured text,
	// writeable only via MemGPT function calls.
	WorkingContext string

	HistoricalContext string

	// The system instructions are readonly (static) and contain information
	// on the MemGPT control flow, the intended usage of the different memory
	// levels, and instructions on how to use the MemGPT functions
	// (e.g. how to retrieve out-of-context data).
	SystemInstructions map[string]string

	// Intarface for perfomring operations on the data storage
	Storage MemoryStorage
}

// MemoryContext can be viewd as state, or core memory with
// perisistance through a db of any kind,

// The MemoryContext needs to be able to load previous conversations
// internal messages, and context into it's state by rebuilding it
// more persisttance storage and archival storage.

// When chat history queue is full, the oldest messages are evicted
// from the queue and summerized into current working context.

// core memory - fixed size memory context with its state saved in persistance storage db
// archive memory - unlimited size db storage for all messages and contexts

func NewMemoryContext(storage MemoryStorage) *MemoryContext {
	return &MemoryContext{
		Storage: storage,
		SystemInstructions: map[string]string{
			"assistant":                        PrimerTemplate,
			"memoryPressure:WorkingContext":    MemoryPressureWorkingContext,
			"memoryPressure:HistoricalContext": MemoryPressureHistoricalContext,
			"memoryPressure:ChatHistory":       MemoryPressureChatHistory,
			"memoryPressure:Messages":          MemoryPressureMessages,
		},
	}
}

// Load all the messages and context from core memory
func (memory *MemoryContext) Load() error {
	msgs, err := memory.Storage.LoadMessages()
	if err != nil {
		return err
	}

	chatHistory, err := memory.Storage.LoadChatHistory()
	if err != nil {
		return err
	}

	workingContext, err := memory.Storage.LoadWorkingContext()
	if err != nil {
		return err
	}

	historicalContext, err := memory.Storage.LoadHistoricalContext()
	if err != nil {
		return err
	}

	memory.Messages = msgs
	memory.ChatHistory = chatHistory
	memory.WorkingContext = workingContext
	memory.HistoricalContext = historicalContext

	return nil
}

// Save current moemory context state to core memory
func (memory *MemoryContext) Save() error {

	err := memory.Storage.SaveMessages(memory.Messages)
	if err != nil {
		return err
	}

	err = memory.Storage.SaveChatHistory(memory.ChatHistory)
	if err != nil {
		return err
	}

	err = memory.Storage.SaveWorkingContext(memory.WorkingContext)
	if err != nil {
		return err
	}

	err = memory.Storage.SaveHistoricalContext(memory.HistoricalContext)
	if err != nil {
		return err
	}

	return nil
}

// Summarize or compress memories into working context or historical context
// to save space
func (memory *MemoryContext) Compress(args string) error {
	// inputs are working and historical Summary generated
	// by llm, based on chat history and messsages
	// updates memory.WorkingContext and memory.HistoricalContext
	// saves the memory context state to core memory

	var input struct {
		WorkingContextSummary    string `json:"workingContextSummary"`
		HistoricalContextSummary string `json:"historicalContextSummary"`
	}

	err := json.Unmarshal([]byte(args), &input)
	if err != nil {
		return err
	}

	memory.WorkingContext = input.WorkingContextSummary
	memory.HistoricalContext = input.HistoricalContextSummary

	err = memory.Storage.SaveWorkingContext(memory.WorkingContext)
	if err != nil {
		return err
	}

	err = memory.Storage.SaveHistoricalContext(memory.HistoricalContext)
	if err != nil {
		return err
	}

	return nil
}

// Move infromation from core memory to archive memory
func (memory *MemoryContext) Memorize(args string) error {
	// can happen when chat history is full
	// save chat history msgs to archive storage
	// removes overflowing messages in chat history
	// saves the new summary to core memory and archive memory
	// input should be the summary of the flushed chat history messages

	var input struct {
		Summary string `json:"summary"`
	}

	err := json.Unmarshal([]byte(args), &input)
	if err != nil {
		return err
	}

	err = memory.FlushMessages()
	if err != nil {
		return err
	}

	memory.WorkingContext = input.Summary

	err = memory.Storage.SaveWorkingContext(memory.WorkingContext)
	if err != nil {
		return err
	}

	err = memory.Storage.ArchiveWorkingContext(memory.WorkingContext)
	if err != nil {
		return err
	}

	return nil
}

// Generate internal thoughts about the context
func (memory *MemoryContext) Reflect(args string) error {
	// can happen when messages is full
	// save messages to archive storage
	// removes overflowing messages in messages
	// saves the new summary to core memory and archive memory
	// input should be the summary of the flushed messages

	var input struct {
		Summary string `json:"summary"`
	}

	err := json.Unmarshal([]byte(args), &input)
	if err != nil {
		return err
	}

	err = memory.FlushMessages()
	if err != nil {
		return err
	}

	memory.HistoricalContext = input.Summary

	err = memory.Storage.SaveHistoricalContext(memory.HistoricalContext)
	if err != nil {
		return err
	}

	err = memory.Storage.ArchiveHistoricalContext(memory.HistoricalContext)
	if err != nil {
		return err
	}

	return nil
}

func (memory *MemoryContext) Archive() error {
	err := memory.Storage.ArchiveWorkingContext(memory.WorkingContext)
	if err != nil {
		return err
	}

	err = memory.Storage.ArchiveHistoricalContext(memory.HistoricalContext)
	if err != nil {
		return err
	}

	err = memory.Storage.ArchiveChatHistory(memory.ChatHistory)
	if err != nil {
		return err
	}

	err = memory.Storage.ArchiveMessages(memory.Messages)
	if err != nil {
		return err
	}

	return nil
}

// Recall information from archive storage
func (memory *MemoryContext) Recall() error {
	workingContext, err := memory.Storage.RecallWorkingContext()
	if err != nil {
		return err
	}

	historicalContext, err := memory.Storage.RecallHistoricalContext()
	if err != nil {
		return err
	}

	msgs, err := memory.Storage.RecallMessages()
	if err != nil {
		return err
	}

	chatHistory, err := memory.Storage.RecallChatHistory()
	if err != nil {
		return err
	}

	memory.Messages = msgs
	memory.ChatHistory = chatHistory
	memory.WorkingContext = workingContext
	memory.HistoricalContext = historicalContext

	return nil
}

func (memory *MemoryContext) Search() error {
	// use similarity search to recall information from archive storage
	// input will be tbd

	return nil
}

func (memory *MemoryContext) FlushMessages() error {
	// while the token size of all messages in buffer
	// is equal or greater than the max token size  minus the current input msg(?)
	// keep removing messages from messages buffer

	return nil

}

func (memory *MemoryContext) FlushChatHistory() error {
	// while the token size of all chat history messages in buffer
	// is equal or greater than the max token size  minus the last input msg(?)
	// keep removing messages from chat history buffer

	return nil
}

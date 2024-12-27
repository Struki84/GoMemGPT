package memory

import "github.com/tmc/langchaingo/llms"

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
)

type MemoryStorage interface {
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

func NewMemoryContext(storage MemoryStorage) *MemoryContext {
	return &MemoryContext{
		Storage: storage,
		SystemInstructions: map[string]string{
			"assistant": PrimerTemplate,
		},
	}
}

// Add new messages to core memory
func (memory *MemoryContext) SaveMessages() error {
	return nil
}

func (memory *MemoryContext) LoadMessages() ([]llms.MessageContent, error) {
	return []llms.MessageContent{}, nil
}

func (memory *MemoryContext) Memorize() {}
func (memory *MemoryContext) Recall()   {}
func (memory *MemoryContext) Reflect()  {}
func (memory *MemoryContext) Compress() {}

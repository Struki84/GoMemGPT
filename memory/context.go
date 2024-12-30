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
			"assistant": PrimerTemplate,
		},
	}
}

// Load all the messages and context from core memory
func (memory *MemoryContext) Load() error {
	return nil
}

// Save current moemory context state to core memory
func (memory *MemoryContext) Save() error {
	return nil
}

// Summarize or compress memories into working context or historical context
// to save space
func (memory *MemoryContext) Compress(workingContextSummary, historicalContextSummary string) error {
	// inputs are working and historical Summary generated
	// by llm, based on chat history and messsages

	// updates memory.WorkingContext and memory.HistoricalContext

	// saves the memory context state to core memory
	return nil
}

// Move infromation from core memory to archive memory
func (memory *MemoryContext) Memorize(summary string) error {
	// can happen when chat history is full
	// save chat history msgs to archive storage
	// removes overflowing messages in chat history
	// saves the new summary to core memory and archive memory
	// input should be the summary of the flushed chat history messages

	return nil
}

// Generate internal thoughts about the context
func (memory *MemoryContext) Reflect(summary string) error {
	// can happen when messages is full
	// save messages to archive storage
	// removes overflowing messages in messages
	// saves the new summary to core memory and archive memory
	// input should be the summary of the flushed messages

	return nil
}

// Recall information from archive storage
func (memory *MemoryContext) Recall() {
	// use similarity search to recal information from archive storage
	// input will be tbd
}

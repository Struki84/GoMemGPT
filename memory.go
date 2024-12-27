package main

import (
	"github.com/tmc/langchaingo/llms"
)

var (
	DefaultContextSize     int     = 1024
	DefaultWorkingCtxSieze float32 = 0.25
	DefaultMsgsSize        float32 = 0.75
)

type MemoryStorage interface {
}

// Main context
type MemoryContext struct {
	// FIFO Message Queue, stores a rolling history of messages,
	// including  messages between the agent and user, as well as system
	// messages (e.g. memory warnings) and function call inputs
	// and outputs. The first index in the FIFO queue stores a system
	// message containing a recursive summary of messages that have
	// been evicted from the queue.
	Messages []llms.MessageContent

	// Current working context
	// Working context is a fixed-size read/write block of unstructured text,
	// writeable only via MemGPT function calls.
	WorkingContext string

	// The system instructions are readonly (static) and contain information
	// on the MemGPT control flow, the intended usage of the different memory
	// levels, and instructions on how to use the MemGPT functions
	// (e.g. how to retrieve out-of-context data).
	SystemInstructions map[string]string

	// Intarface for perfomring operations on the data storage
	Storage MemoryStorage
}

func (memory *MemoryContext) SaveConversation() {}
func (memory *MemoryContext) LoadConversation() {}
func (memory *MemoryContext) Memorize()         {}
func (memory *MemoryContext) Recall()           {}
func (memory *MemoryContext) Reflect()          {}
func (memory *MemoryContext) Compress()         {}

// this is my queue manager - The queue manager manages messages in recall storage
// and the FIFO queue.
type MemoryManager struct {
	llm            llms.Model
	mainContext    *MemoryContext
	maxContextSize int
	workingCtxSize int
	msgsSize       int
}

func NewMemoryManager(llm llms.Model, storage MemoryStorage) *MemoryManager {
	return &MemoryManager{
		maxContextSize: DefaultContextSize,
		workingCtxSize: int(float32(DefaultContextSize) * DefaultWorkingCtxSieze),
		msgsSize:       int(float32(DefaultContextSize) * DefaultMsgsSize),
	}
}

func (cm *MemoryManager) LoadInstructions() llms.MessageContent {
	return llms.TextParts(llms.ChatMessageTypeSystem, cm.mainContext.SystemInstructions["assistant"])
}

func (cm *MemoryManager) Update(msg []llms.MessageContent) error {
	// Main message processing pipeline
	// 1. Queue management
	// 2. Context window management
	// 3. LLM processing
	// 4. Function execution
	// 5. Memory updates

	return nil
}

func (cm *MemoryManager) FlushMessages() error {
	// Implement memory pressure handling
	// - Check context window usage
	// - Trigger eviction if needed
	// - Update working context
	return nil
}

// this is my function executor / tool node
// executes llm functions and interactgs with
// archival and recall storage
type StorageManager struct {
	mainContext *MemoryContext
	functions   []llms.FunctionDefinition
}

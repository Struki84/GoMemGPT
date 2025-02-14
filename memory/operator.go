package memory

import (
	"errors"
	"fmt"
	"log"

	"github.com/pkoukk/tiktoken-go"
	"github.com/tmc/langchaingo/llms"
)

type MemoryOperator struct {
	MainContext *MemoryContext
	Storage     MemoryStorage
}

func NewMemoryOperator(mainContext *MemoryContext) *MemoryOperator {
	return &MemoryOperator{
		MainContext: mainContext,
		Storage:     mainContext.Storage,
	}
}

func (operator MemoryOperator) Load() error {
	msgs, err := operator.Storage.LoadMessages()
	if err != nil {
		return err
	}

	workingContext, err := operator.Storage.LoadWorkingContext()
	if err != nil {
		return err
	}

	operator.MainContext.Messages = msgs
	operator.MainContext.WorkingContext = workingContext

	// log.Printf("Messages loaded: %v", operator.MainContext.Messages)

	return nil
}

// Save current memory context state to core memory
func (operator MemoryOperator) Save() error {

	err := operator.Storage.SaveMessages(operator.MainContext.Messages)
	if err != nil {
		return err
	}

	err = operator.Storage.SaveWorkingContext(operator.MainContext.WorkingContext)
	if err != nil {
		return err
	}

	return nil
}

func (operator MemoryOperator) Reflect(summary string) error {
	// inputs is working contex. Summary generated
	// by llm based on all the current messsages in context

	operator.MainContext.WorkingContext = summary

	err := operator.Storage.SaveWorkingContext(operator.MainContext.WorkingContext)
	if err != nil {
		return err
	}

	return nil
}

// Move infromation from core memory to archive memory
func (operator MemoryOperator) Memorize(summary string) error {
	// can happen when chat history is full
	// save chat history msgs to archive storage
	// removes overflowing messages in chat history
	// saves the new summary to core memory and archive memory
	// input should be the summary of the flushed chat history messages

	// we flush all the messages from shrot term memory and leave only the last 3
	// the evicted messages are appended to long term memory
	// this is probably a temp solution
	// clanedMsgs := operator.MainContext.Messages[max(0, len(operator.MainContext.Messages)-3):]

	// err = operator.Storage.SaveMessages(clanedMsgs)
	// if err != nil {
	// 	return err
	// }

	// operator.MainContext.Messages = clanedMsgs
	err := operator.Storage.SaveMessages(operator.MainContext.Messages)
	if err != nil {
		log.Printf("Error saving messages: %v", err)
		return err
	}

	err = operator.Storage.ArchiveMessages(operator.MainContext.Messages)
	if err != nil {
		log.Printf("Error archiving messages: %v", err)
		return err
	}

	err = operator.Storage.SaveWorkingContext(summary)
	if err != nil {
		log.Printf("Error saving working context: %v", err)
		return err
	}

	msgs, err := operator.Storage.LoadMessages()
	if err != nil {
		log.Printf("Error loading messages: %v", err)
		return err
	}

	workingCtx, err := operator.Storage.LoadWorkingContext()
	if err != nil {
		log.Printf("Error loading working context: %v", err)
		return err
	}

	operator.MainContext.Messages = msgs
	operator.MainContext.WorkingContext = workingCtx

	return nil
}

func (operator MemoryOperator) Recall(query string, limit, page int) error {
	msgs, err := operator.Storage.RecallMessages(query, limit, page)
	if err != nil {
		return err
	}

	encoder, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		log.Printf("Error creating tiktoken encoder: %v", err)
		return errors.New(fmt.Sprintf("error creating tiktoken encoder: %v", err))
	}

	msgSize := len(encoder.Encode(msgs, nil, nil))

	if operator.MainContext.CurrentMessagesSize()+msgSize < int(operator.MainContext.msgsSize*0.9) {
		chatHistory := llms.TextParts(llms.ChatMessageTypeSystem, msgs)
		operator.MainContext.Messages = append(operator.MainContext.Messages, chatHistory)

		return nil
	}

	return errors.New("Memory overflow: request less messages per page or clear your memory")
}

func (operator MemoryOperator) Think(thought string) string {
	return thought
}

func (operator MemoryOperator) InternalOutput(msg string) string {
	return msg
}

func (operator MemoryOperator) ExternalOutput(msg string) string {
	return msg
}

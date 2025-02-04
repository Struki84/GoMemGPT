package memory

type MemoryOperator struct {
	MainContext *MemoryContext
	Storage     MemoryStorage
}

func NewMemoryOperator(mainContext *MemoryContext) *MemoryOperator {
	return &MemoryOperator{
		MainContext: mainContext,
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
	clanedMsgs := operator.MainContext.Messages[max(0, len(operator.MainContext.Messages)-3):]

	err := operator.Storage.ArchiveMessages(operator.MainContext.Messages)
	if err != nil {
		return err
	}

	err = operator.Storage.SaveMessages(clanedMsgs)
	if err != nil {
		return err
	}

	operator.MainContext.Messages = clanedMsgs

	err = operator.Storage.SaveWorkingContext(summary)
	if err != nil {
		return err
	}

	operator.MainContext.WorkingContext = summary

	return nil
}

func (operator MemoryOperator) Recall() error {
	msgs, err := operator.Storage.RecallMessages()
	if err != nil {
		return err
	}

	operator.MainContext.Messages = msgs

	return nil
}

func (operator MemoryOperator) InternalOutput(msg string) string {
	return msg
}

func (operator MemoryOperator) ExternalOutput(msg string) string {
	return msg
}

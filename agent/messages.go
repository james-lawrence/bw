package agent

// MessagesToPtr convert a set of messages to a set of pointers to messages.
func MessagesToPtr(messages ...Message) []*Message {
	result := make([]*Message, 0, len(messages))
	for _, m := range messages {
		dup := m
		result = append(result, &dup)
	}
	return result
}

// MessagesFromPtr converts a set of pointers to messages to a set of messages.
func MessagesFromPtr(messages ...*Message) []Message {
	result := make([]Message, 0, len(messages))
	for _, m := range messages {
		result = append(result, *m)
	}
	return result
}

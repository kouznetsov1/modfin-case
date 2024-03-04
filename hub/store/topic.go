package store

type TopicStore struct {
	topics map[string]struct{}
}

func NewTopicStore() *TopicStore {
	return &TopicStore{topics: make(map[string]struct{})}
}

func (ts *TopicStore) AddTopic(topic string) {
	ts.topics[topic] = struct{}{}
}

func (ts *TopicStore) RemoveTopic(topic string) {
	delete(ts.topics, topic)
}

func (ts *TopicStore) GetTopics() []string {
	topics := make([]string, 0, len(ts.topics))
	for topic := range ts.topics {
		topics = append(topics, topic)
	}
	return topics
}

func (ts *TopicStore) HasTopic(topic string) bool {
	_, ok := ts.topics[topic]
	return ok
}

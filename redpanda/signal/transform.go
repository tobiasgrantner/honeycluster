package main

import (
	"encoding/json"
	"strings"

	"github.com/redpanda-data/redpanda/src/transform-sdk/go/transform"
)

var (
	keys = make(map[string]bool)
)

func main() {
	// Register your transform function.
	// This is a good place to perform other setup too.
	keys = make(map[string]bool)
	transform.OnRecordWritten(doTransform)
}

var fieldsToRemove = []string{"pid", "tid", "auid", "uid", "exec_id", "parent_exec_id"}

type Message struct {
	Timestamp string                 `json:"timestamp"`
	Data      map[string]interface{} `json:",inline"`
}

func removeTimeFields(obj interface{}) {
	switch v := obj.(type) {
	case map[string]interface{}:
		for key, value := range v {
			// If the key contains "time", delete it
			for _, field := range fieldsToRemove {
				if strings.EqualFold(key, field) {
					delete(v, key)
					break
				}
			}
			if strings.Contains(strings.ToLower(key), "time") {
				delete(v, key)
			} else {
				// If the value is a map or a slice, recursively remove time fields
				removeTimeFields(value)
			}
		}
	case []interface{}:
		for i := range v {
			removeTimeFields(v[i])
		}
	}
}

func doTransform(e transform.WriteEvent, w transform.RecordWriter) error {
	// Unmarshal the incoming message into a map
	var incomingMessage map[string]interface{}
	err := json.Unmarshal(e.Record().Value, &incomingMessage)
	if err != nil {
		return err
	}

	// Remove the time fields from the message
	removeTimeFields(incomingMessage)
	// Extract 3 fields from the JSON and concat them as key
	key := e.Record().Key

	_, seen := keys[string(key)]
	if !seen {
		keys[string(key)] = true

		// Marshal the result back to JSON
		jsonData, err := json.Marshal(incomingMessage)
		if err != nil {
			return err
		}

		// Create a new record with the JSON data
		record := &transform.Record{
			Key:       []byte(key),
			Value:     jsonData,
			Offset:    e.Record().Offset,
			Timestamp: e.Record().Timestamp,
			Headers:   e.Record().Headers,
		}

		// Write the record to the destination topic
		err = w.Write(*record)
		if err != nil {
			return err
		}

	}

	return nil
}

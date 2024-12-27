package browser

import (
	"encoding/json"
	"io/ioutil"
	"time"
)

// RecordedStep represents a recorded browser action
type RecordedStep struct {
	Type      string                 `json:"type"`
	Params    map[string]interface{} `json:"params"`
	Timestamp time.Time              `json:"timestamp"`
}

// Recorder records browser actions
type Recorder struct {
	steps []RecordedStep
}

func NewRecorder() *Recorder {
	return &Recorder{
		steps: make([]RecordedStep, 0),
	}
}

func (r *Recorder) Record(stepType string, params map[string]interface{}) {
	r.steps = append(r.steps, RecordedStep{
		Type:      stepType,
		Params:    params,
		Timestamp: time.Now(),
	})
}

func (r *Recorder) ExportSequence(name string) *AutomationSequence {
	steps := make([]AutomationStep, len(r.steps))
	for i, step := range r.steps {
		steps[i] = AutomationStep{
			Type:   step.Type,
			Params: step.Params,
		}
	}

	return &AutomationSequence{
		Name:  name,
		Steps: steps,
	}
}

// SaveToFile saves the recorded sequence to a file
func (r *Recorder) SaveToFile(filename string) error {
	sequence := r.ExportSequence("Recorded Sequence")
	data, err := json.MarshalIndent(sequence, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0644)
}

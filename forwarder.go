package hypertrace

import (
	"encoding/json"
	"fmt"
)

type IForwarder interface {
	ForwardTraceData(UID string, data []*TraceData) error
}

type StdOutForwarder struct {
}

func (forwarder *StdOutForwarder) ForwardTraceData(UID string, data []*TraceData) error {
	fmt.Printf("Forward data for %s", UID)
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	fmt.Println(string(jsonBytes))
	return nil
}
package gokautil

import "encoding/json"

type JSONCodec[T any] struct{}

func (c JSONCodec[T]) Encode(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

func (c JSONCodec[T]) Decode(data []byte) (interface{}, error) {
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	return &value, nil
}

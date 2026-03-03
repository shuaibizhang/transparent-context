package codec

import (
	"encoding/json"
	"google.golang.org/grpc/encoding"
)

const Name = "json"

func init() {
	encoding.RegisterCodec(codec{})
}

type codec struct{}

func (codec) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (codec) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (codec) Name() string {
	return Name
}

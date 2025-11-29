package iso8601duration

import (
	"bytes"
	"encoding"
	"encoding/json"
)

// 型チェック
var (
	_ encoding.TextMarshaler     = Duration{}
	_ encoding.TextUnmarshaler   = (*Duration)(nil)
	_ encoding.BinaryMarshaler   = Duration{}
	_ encoding.BinaryUnmarshaler = (*Duration)(nil)
	_ json.Marshaler             = Duration{}
	_ json.Unmarshaler           = (*Duration)(nil)
)

var nullBytes = []byte("null")

func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

func (d *Duration) UnmarshalText(data []byte) error {
	t, err := ParseString(string(data))
	if err != nil {
		return err
	}
	*d = t
	return nil
}

func (d Duration) MarshalBinary() ([]byte, error) {
	return d.MarshalText()
}

func (d *Duration) UnmarshalBinary(data []byte) error {
	return d.UnmarshalText(data)
}

func (d Duration) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	s := d.String()
	err := enc.Encode(s)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, nullBytes) {
		*d = Duration{}
		return nil
	}

	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		data = data[1 : len(data)-1]
	}
	t, err := ParseString(string(data))
	if err != nil {
		return err
	}
	*d = t
	return nil
}

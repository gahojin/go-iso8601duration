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

func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

func (d *Duration) UnmarshalText(data []byte) error {
	t, err := ParseString(string(data))
	if err == nil {
		if t == nil {
			return ErrBadFormat
		}
		*d = *t
	}
	return err
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
	dec := json.NewDecoder(bytes.NewBuffer(data))
	var s string
	if err := dec.Decode(&s); err != nil {
		return err
	}
	t, err := ParseString(s)
	if err == nil {
		if t == nil {
			return ErrBadFormat
		}
		*d = *t
	}
	return err
}

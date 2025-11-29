package iso8601duration

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

type dummyStruct struct {
	Value    Duration
	Nullable *Duration
	Empty    Duration `json:",omitzero"`
}

func TestTextMarshal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		expect := Duration{
			Negative:    rapid.Bool().Draw(t, "negative"),
			Years:       rapid.Uint32().Draw(t, "years"),
			Months:      rapid.Uint32().Draw(t, "months"),
			Weeks:       rapid.Uint32().Draw(t, "weeks"),
			Days:        rapid.Uint32().Draw(t, "days"),
			Hours:       rapid.Uint32().Draw(t, "hours"),
			Minutes:     rapid.Uint32().Draw(t, "minutes"),
			Seconds:     rapid.Uint32().Draw(t, "seconds"),
			Nanoseconds: rapid.Uint32().Draw(t, "nanoseconds"),
		}

		data, err := expect.MarshalText()
		assert.NoError(t, err)
		assert.NotNil(t, data)

		var actual Duration
		err = actual.UnmarshalText(data)
		assert.NoError(t, err)
		// ナノ秒のうち、秒単位の桁は、秒に加算する
		expect.Seconds += uint32(time.Duration(expect.Nanoseconds) / time.Second)
		expect.Nanoseconds = uint32(time.Duration(expect.Nanoseconds) % time.Second)
		assert.Equal(t, expect, actual)
	})
}

func TestBinaryMarshal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		expect := Duration{
			Negative:    rapid.Bool().Draw(t, "negative"),
			Years:       rapid.Uint32().Draw(t, "years"),
			Months:      rapid.Uint32().Draw(t, "months"),
			Weeks:       rapid.Uint32().Draw(t, "weeks"),
			Days:        rapid.Uint32().Draw(t, "days"),
			Hours:       rapid.Uint32().Draw(t, "hours"),
			Minutes:     rapid.Uint32().Draw(t, "minutes"),
			Seconds:     rapid.Uint32().Draw(t, "seconds"),
			Nanoseconds: rapid.Uint32().Draw(t, "nanoseconds"),
		}
		sut := dummyStruct{
			Value: expect,
		}
		buf := bytes.NewBuffer(nil)
		err := gob.NewEncoder(buf).Encode(&sut)
		assert.NoError(t, err)
		data := buf.Bytes()
		assert.NotNil(t, data)

		var actual dummyStruct
		err = gob.NewDecoder(bytes.NewBuffer(data)).Decode(&actual)
		assert.NoError(t, err)
		// ナノ秒のうち、秒単位の桁は、秒に加算する
		expect.Seconds += uint32(time.Duration(expect.Nanoseconds) / time.Second)
		expect.Nanoseconds = uint32(time.Duration(expect.Nanoseconds) % time.Second)
		assert.Equal(t, expect, actual.Value)
	})
}

func TestJSONMarshal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		expect := dummyStruct{
			Value: Duration{
				Negative:    rapid.Bool().Draw(t, "negative"),
				Years:       rapid.Uint32().Draw(t, "years"),
				Months:      rapid.Uint32().Draw(t, "months"),
				Weeks:       rapid.Uint32().Draw(t, "weeks"),
				Days:        rapid.Uint32().Draw(t, "days"),
				Hours:       rapid.Uint32().Draw(t, "hours"),
				Minutes:     rapid.Uint32().Draw(t, "minutes"),
				Seconds:     rapid.Uint32().Draw(t, "seconds"),
				Nanoseconds: rapid.Uint32().Draw(t, "nanoseconds"),
			},
		}

		data, err := json.Marshal(expect)
		assert.NoError(t, err)
		assert.NotNil(t, data)

		var actual dummyStruct
		err = json.Unmarshal(data, &actual)
		assert.NoError(t, err)
		// ナノ秒のうち、秒単位の桁は、秒に加算する
		expect.Value.Seconds += uint32(time.Duration(expect.Value.Nanoseconds) / time.Second)
		expect.Value.Nanoseconds = uint32(time.Duration(expect.Value.Nanoseconds) % time.Second)
		assert.Equal(t, expect, actual)
	})

	var actual dummyStruct
	err := json.Unmarshal([]byte(`{"Empty":null}`), &actual)
	assert.NoError(t, err)

}

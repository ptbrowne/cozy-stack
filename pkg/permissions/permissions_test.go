package permissions

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerbToString(t *testing.T) {
	vs := Verbs(GET, DELETE)
	assert.Equal(t, "GET,DELETE", vs.String())

	vs3 := ALL
	assert.Equal(t, "ALL", vs3.String())

	vs4 := VerbSplit("ALL")
	assert.Equal(t, "ALL", vs4.String())

}

func TestRuleToJSON(t *testing.T) {
	r := Rule{
		Type:  "io.cozy.contacts",
		Verbs: Verbs(GET, POST),
	}

	b, err := json.Marshal(r)
	assert.NoError(t, err)
	assert.Equal(t, `{"type":"io.cozy.contacts","verbs":["GET","POST"]}`, string(b))
}

func TestSetToJSON(t *testing.T) {
	s := Set{
		Rule{
			Title:       "contacts",
			Description: "Required for autocompletion on @name",
			Type:        "io.cozy.contacts",
			Verbs:       Verbs(GET),
		},
		Rule{
			Title:       "images",
			Description: "Required for the background",
			Type:        "io.cozy.files",
			Verbs:       Verbs(GET),
			Values:      []string{"io.cozy.files.music-dir"},
		},
		Rule{
			Title:       "mail",
			Description: "Required to send a congratulations email to your friends",
			Type:        "io.cozy.jobs",
			Selector:    "worker",
			Values:      []string{"sendmail"},
		},
	}

	b, err := json.Marshal(s)
	assert.NoError(t, err)
	assertEqualJSON(t, b, `{
    "contacts": {
      "type": "io.cozy.contacts",
      "description": "Required for autocompletion on @name",
      "verbs": ["GET"]
    },
    "images": {
      "type": "io.cozy.files",
      "description": "Required for the background",
      "verbs": ["GET"],
      "values": ["io.cozy.files.music-dir"]
    },
    "mail": {
      "type": "io.cozy.jobs",
      "description": "Required to send a congratulations email to your friends",
      "selector": "worker",
      "values": ["sendmail"]
    }
  }`)

}

func TestJSON2Set(t *testing.T) {
	jsonSet := []byte(`{
    "contacts": {
      "type": "io.cozy.contacts",
      "description": "Required for autocompletion on @name",
      "verbs": ["GET"]
    },
    "images": {
      "type": "io.cozy.files",
      "description": "Required for the background",
      "verbs": ["GET"],
      "values": ["io.cozy.files.music-dir"]
    },
    "mail": {
      "type": "io.cozy.jobs",
      "description": "Required to send a congratulations email to your friends",
      "selector": "worker",
      "values": ["sendmail"]
    }
  }`)
	var s Set
	err := json.Unmarshal(jsonSet, &s)
	assert.NoError(t, err)
}

func TestSetToString(t *testing.T) {
	s := Set{
		Rule{
			Title:       "contacts",
			Description: "Required for autocompletion on @name",
			Type:        "io.cozy.contacts",
		},
		Rule{
			Title:       "images",
			Description: "Required for the background",
			Type:        "io.cozy.files",
			Verbs:       Verbs(GET),
			Values:      []string{"io.cozy.files.music-dir"},
		},
		Rule{
			Title:    "sendmail",
			Type:     "io.cozy.jobs",
			Selector: "worker",
			Values:   []string{"sendmail"},
		},
	}

	out, err := s.MarshalScopeString()
	assert.NoError(t, err)
	assert.Equal(t, out, "io.cozy.contacts io.cozy.files:GET:io.cozy.files.music-dir io.cozy.jobs:ALL:sendmail:worker")
}

func TestStringToSet(t *testing.T) {

	_, err := UnmarshalRuleString("")
	assert.Error(t, err)

	_, err = UnmarshalRuleString("type:verb:selec:value:wtf")
	assert.Error(t, err)

	s, err := UnmarshalScopeString("io.cozy.contacts io.cozy.files:GET:io.cozy.files.music-dir")

	assert.NoError(t, err)
	assert.Len(t, s, 2)
	assert.Equal(t, "io.cozy.contacts", s[0].Type)
	assert.Equal(t, "io.cozy.files", s[1].Type)
	assert.Len(t, s[1].Verbs, 1)
	assert.Equal(t, Verbs(GET), s[1].Verbs)
	assert.Len(t, s[1].Values, 1)
	assert.Equal(t, "io.cozy.files.music-dir", s[1].Values[0])

	rule, err := UnmarshalRuleString("io.cozy.events:GET:mygreatcalendar,othercalendar:calendar-id")
	assert.NoError(t, err)
	assert.Equal(t, "io.cozy.events", rule.Type)
	assert.Equal(t, Verbs(GET), rule.Verbs)
	assert.Len(t, rule.Values, 2)
	assert.Equal(t, "mygreatcalendar", rule.Values[0])
	assert.Equal(t, "othercalendar", rule.Values[1])
	assert.Equal(t, "calendar-id", rule.Selector)

}

func TestAllowType(t *testing.T) {
	s := Set{Rule{Type: "io.cozy.contacts"}}
	assert.True(t, s.Allow(GET, &validable{doctype: "io.cozy.contacts"}))
	assert.True(t, s.Allow(DELETE, &validable{doctype: "io.cozy.contacts"}))
	assert.False(t, s.Allow(GET, &validable{doctype: "io.cozy.files"}))
}

func TestAllowVerbs(t *testing.T) {
	s := Set{Rule{Type: "io.cozy.contacts", Verbs: Verbs(GET)}}
	assert.True(t, s.Allow(GET, &validable{doctype: "io.cozy.contacts"}))
	assert.False(t, s.Allow(DELETE, &validable{doctype: "io.cozy.contacts"}))
	assert.False(t, s.Allow(GET, &validable{doctype: "io.cozy.files"}))
}

func TestAllowValues(t *testing.T) {
	s := Set{Rule{
		Type:   "io.cozy.contacts",
		Values: []string{"id1"},
	}}
	assert.True(t, s.Allow(POST, &validable{doctype: "io.cozy.contacts", id: "id1"}))
	assert.False(t, s.Allow(POST, &validable{doctype: "io.cozy.contacts", id: "id2"}))
}

func TestAllowValuesSelector(t *testing.T) {
	s := Set{Rule{
		Type:     "io.cozy.contacts",
		Selector: "foo",
		Values:   []string{"bar"},
	}}
	assert.True(t, s.Allow(GET, &validable{
		doctype: "io.cozy.contacts",
		values:  map[string]string{"foo": "bar"}}))

	assert.False(t, s.Allow(GET, &validable{
		doctype: "io.cozy.contacts",
		values:  map[string]string{"foo": "baz"}}))
}

func TestAllowWholeType(t *testing.T) {
	s := Set{Rule{Type: "io.cozy.contacts", Verbs: Verbs(GET)}}
	assert.True(t, s.AllowWholeType(GET, "io.cozy.contacts"))

	s2 := Set{Rule{Type: "io.cozy.contacts", Values: []string{"id1"}}}
	assert.False(t, s2.AllowWholeType(GET, "io.cozy.contacts"))
}

func TestAllowID(t *testing.T) {
	s := Set{Rule{Type: "io.cozy.contacts"}}
	assert.True(t, s.AllowID(GET, "io.cozy.contacts", "id1"))

	s2 := Set{Rule{Type: "io.cozy.contacts", Values: []string{"id1"}}}
	assert.True(t, s2.AllowID(GET, "io.cozy.contacts", "id1"))

	s3 := Set{Rule{Type: "io.cozy.contacts", Selector: "foo", Values: []string{"bar"}}}
	assert.False(t, s3.AllowID(GET, "io.cozy.contacts", "id1"))
}

func TestAllowCustomType(t *testing.T) {

	s := Set{Rule{Type: "io.cozy.files", Selector: "path", Values: []string{"/testp/"}}}

	y := &validableFile{"/testp/test"}
	n := &validableFile{"/not-testp/test"}

	assert.True(t, s.Allow(GET, y))
	assert.False(t, s.Allow(GET, n))
}

func assertEqualJSON(t *testing.T, value []byte, expected string) {
	expectedBytes := new(bytes.Buffer)
	err := json.Compact(expectedBytes, []byte(expected))
	assert.NoError(t, err)
	assert.Equal(t, expectedBytes.String(), string(value))
}

type validable struct {
	id      string
	doctype string
	values  map[string]string
}

func (t *validable) ID() string             { return t.id }
func (t *validable) DocType() string        { return t.doctype }
func (t *validable) Valid(f, e string) bool { return t.values[f] == e }

type validableFile struct {
	path string
}

func (t *validableFile) ID() string      { return t.path }
func (t *validableFile) DocType() string { return "io.cozy.files" }
func (t *validableFile) Valid(f, e string) bool {
	return f == "path" && strings.HasPrefix(t.path, e)
}

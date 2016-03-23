package rserve

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

var testPoints = datapoints([]Datapoint{
	{1456080911, 6.24},
	{1456080851, 6.23},
	{1456080971, 6.39},
})

func TestDatapointMarshalJSON(t *testing.T) {
	d := Datapoint{Timestamp: 1456080911, Value: 1902.3385}

	b, err := d.MarshalJSON()
	if err != nil {
		t.Error(err)
	}

	exp := string("[1456080911,1902.3385]")
	if string(b) != exp {
		t.Errorf("expected '%s', got '%s'", exp, b)
	}
}

func TestDatapointUnmarshalJSON(t *testing.T) {
	b := []byte("[1456080911,1902.3385]")
	d := Datapoint{}
	dExp := Datapoint{Timestamp: 1456080911, Value: 1902.3385}

	err := json.Unmarshal(b, &d)
	if err != nil {
		t.Error(err)
	}

	if d.Timestamp != dExp.Timestamp || d.Value != dExp.Value {
		t.Errorf("points do not match, '%v' != '%v'", d, dExp)
	}
}

func TestwriteTempCsv(t *testing.T) {
	tname, err := testPoints.writeTempCsv()
	if err != nil {
		t.Errorf("unexpected err: %v", err)
	}

	if strings.Index(tname, "gorp") == -1 {
		t.Error("expected 'gorp' in temp filename")
	}

	_, err = os.Stat(tname)
	if err != nil {
		t.Errorf("stat err: %v", err)
	}
}

func TestwriteCsv(t *testing.T) {
	buf := &bytes.Buffer{}
	err := testPoints.writeCsv(buf)
	if err != nil {
		t.Errorf("unexpected err: %v", err)
	}

	// expect result to be sorted by timestamp
	lines := []string{
		"date,value",
		"2016-02-21 13:54:11,6.23",
		"2016-02-21 13:55:11,6.24",
		"2016-02-21 13:56:11,6.39",
		"", // account for extra newline
	}

	res := strings.Split(buf.String(), "\n")
	if !reflect.DeepEqual(res, lines) {
		t.Errorf("expected:\n%s\ngot:\n%s", lines, res)
	}
}

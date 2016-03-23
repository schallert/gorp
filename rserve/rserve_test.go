package rserve

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"os"
	"testing"

	"github.com/senseyeio/roger"
)

type testClient struct {
	bad bool
}

func (tc *testClient) Eval(command string) (interface{}, error) {
	if tc.bad {
		return nil, errors.New("test error")
	}
	return []byte(command), nil
}

func (tc *testClient) Evaluate(command string) <-chan roger.Packet { return nil }
func (tc *testClient) EvaluateSync(command string) roger.Packet    { return nil }
func (tc *testClient) GetSession() (roger.Session, error)          { return nil, nil }

func newTestClient(bad bool) *client {
	return &client{
		rclient: &testClient{
			bad: bad,
		},
	}
}

func TestNewClient(t *testing.T) {
	errAddrs := [...]string{
		"remote.example.com:6311",
		"localhost",
		"localhost:",
		"localhost:zzz",
		// we expect an error for normal conn too until mock out an rserve
		// daemon :(
		"localhost:6311",
	}

	for _, s := range errAddrs {
		if _, err := NewClient(s); err == nil {
			t.Errorf("Expected error for addr '%s'", s)
		}
	}
}

func TestEval(t *testing.T) {
	var client *client
	client = newTestClient(false)
	b, err := client.eval("test")
	if err != nil {
		t.Errorf("unexpected err: %v", err)
	}

	if string(b.([]byte)) != "test" {
		t.Errorf("expected result 'test': %s", b)
	}

	client = newTestClient(true)
	if _, err := client.eval("test"); err == nil {
		t.Error("expected eval error")
	}
}

func TestProcess(t *testing.T) {
	t.SkipNow()

	var client *client
	client = newTestClient(false)

	res, err := client.process(nil, "ts", "foo")
	if err != nil {
		t.Errorf("unexpected err: %v", err)
	}

	if string(res.PngData) != `processAnomsTs("foo")` {
		t.Errorf("unexpected response: %s", res.PngData)
	}

	client = newTestClient(true)
	if _, err := client.process(nil, "ts", "foo"); err == nil {
		t.Error("expected process error")
	}
}

func TestRCmdString(t *testing.T) {
	if _, err := rCmdString("blah", ""); err == nil {
		t.Error("expected error on method 'blah'")
	}

	for args, exp := range map[[2]string]string{
		{"ts", "/tmp/file.csv"}:   `processAnomsTs("/tmp/file.csv")`,
		{"vec", "/tmp/file2.csv"}: `processAnomsVec("/tmp/file2.csv")`,
	} {
		res, err := rCmdString(args[0], args[1])
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
		if res != exp {
			t.Errorf("expected '%s', got '%s'", exp, res)
		}
	}
}

func TestFormatResponseVec(t *testing.T) {
	response := loadTestData(t, "vec").(vecResponse)

	res, err := formatResponseVec(response)
	if err != nil {
		t.Fatal(err)
	}

	buf := bytes.NewBuffer(res.PngData)
	_, err = png.Decode(buf)
	if err != nil {
		t.Errorf("expected valid png, got: %v", err)
	}

	if len(res.Anomalies) != 86 {
		t.Errorf("test data has 86 anomalies, got %d", len(res.Anomalies))
	}

	if res.Method != "vec" {
		t.Errorf("expected method 'vec', got '%s'", res.Method)
	}
}

func TestFormatResponseTs(t *testing.T) {
	resp := loadTestData(t, "ts").(tsResponse)

	res, err := formatResponseTs(resp)
	if err != nil {
		t.Fatal(err)
	}

	buf := bytes.NewBuffer(res.PngData)
	_, err = png.Decode(buf)
	if err != nil {
		t.Errorf("expected valid png, got: %v", err)
	}

	if len(res.Anomalies) != 86 {
		t.Errorf("test data has 86 anomalies, got %d", len(res.Anomalies))
	}

	if res.Method != "ts" {
		t.Errorf("expected method 'ts', got '%s'", res.Method)
	}
}

func loadTestData(t *testing.T, method string) interface{} {
	f, err := os.Open(fmt.Sprintf("../testdata/r_result_%s.json", method))
	if err != nil {
		t.Fatalf("could not load test data: %v", err)
	}
	defer f.Close()

	switch method {
	case "vec":
		var resp vecResponse
		err = json.NewDecoder(f).Decode(&resp)
		if err != nil {
			t.Fatal(err)
		}
		return resp

	case "ts":
		var resp tsResponse
		err = json.NewDecoder(f).Decode(&resp)
		if err != nil {
			t.Fatal(err)
		}
		return resp

	default:
		t.Fatalf("unrecognized method '%s'", method)
		return nil
	}
}

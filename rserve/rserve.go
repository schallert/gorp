package rserve

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/senseyeio/roger"
)

// The Client interface is implemented by clients that accept an array of
// datapoints and return the corresponding Result.
type Client interface {
	GeneratePNG(points []Datapoint) (Result, error)
}

// client implements Client and talks to a local rserve daemon.
type client struct {
	rclient roger.RClient
}

// A Result represents an array of anomalies (as datapoints), the data for
// the PNG plot, and the method (either "ts" or "vec") that R used to calculate
// the result.
type Result struct {
	Anomalies []Datapoint `json:"anomalies"`
	PngData   []byte      `json:"pngData"`
	Method    string      `json:"method"`
}

// vecResponse is used to decode the JSON-encoded interface{} response from
// roger into a structured format. Vector responses are encoded as an array
// of the anomalous values and an array of the indices those points were
// observed at.
type vecResponse struct {
	Anoms struct {
		Index []int
		Anoms []float64
	}
	Data []byte
}

// tsResponse is used to decode the JSON-encoded interface{} response from
// roger into a structured format. Timeseries responses are encoded as an array
// of anomalous values and an array of the Unix timestamps those values were
// observed at.
type tsResponse struct {
	Anoms struct {
		Anoms     []float64
		Timestamp []float64
	}
	Data []byte
}

// NewClient opens a roger connection to an rserve daemon on a given address
// and returns a client implementing the Client interface.
func NewClient(addr string) (Client, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("splitHostPort err: %v", err)
	}

	if host != "" && host != "localhost" {
		return nil, errors.New("rserve must be running on localhost")
	}

	iport, err := strconv.ParseInt(port, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parseInt err: %v", err)
	}

	rclient, err := roger.NewRClient(host, iport)
	if err != nil {
		return nil, fmt.Errorf("roger err: %v", err)
	}

	rc := client{
		rclient: rclient,
	}

	return &rc, nil
}

func (rc *client) eval(command string) (interface{}, error) {
	res, err := rc.rclient.Eval(command)
	if err != nil {
		return nil, fmt.Errorf("r eval err: %v", err)
	}

	return res, nil
}

// GeneratePNG formats a given array of data points, passes them to the local
// rserve daemon for anomaly detection and returns the structured result.
func (rc *client) GeneratePNG(points []Datapoint) (Result, error) {
	var res Result

	fname, err := datapoints(points).writeTempCsv()
	if err != nil {
		return res, err
	}

	res, err = rc.process(points, "ts", fname)
	if err == nil {
		return res, nil
	}

	res, err = rc.process(points, "vec", fname)
	if err != nil {
		return res, fmt.Errorf("processVec err: %v", err)
	}

	return res, nil
}

func (rc *client) process(points []Datapoint, method, fname string) (Result, error) {
	var res Result

	cmd, err := rCmdString(method, fname)
	if err != nil {
		return res, err
	}

	data, err := rc.eval(cmd)
	if err != nil {
		return res, err
	}

	// this is hacky, but we can avoid manual decoding by encoding the generic
	// response to json and then decoding it in our structured format
	buf := &bytes.Buffer{}
	err = json.NewEncoder(buf).Encode(data)
	if err != nil {
		return res, fmt.Errorf("json encode err: %v", err)
	}

	switch method {
	case "vec":
		var resp vecResponse
		err = json.NewDecoder(buf).Decode(&resp)
		if err != nil {
			return res, fmt.Errorf("vec json decode err: %v", err)
		}
		return formatResponseVec(resp)
	case "ts":
		var resp tsResponse
		err = json.NewDecoder(buf).Decode(&resp)
		if err != nil {
			return res, fmt.Errorf("ts json decode err: %v", err)
		}
		return formatResponseTs(resp)
	default:
		return res, fmt.Errorf("unrecognized method '%s'", method)
	}
}

func formatResponseVec(response vecResponse) (Result, error) {
	var res Result

	ra := response.Anoms
	if len(ra.Index) != len(ra.Anoms) {
		return res, fmt.Errorf("non-equal number of indices and anomalies (%d != %d)", len(ra.Index), len(ra.Anoms))
	}

	anoms := make([]Datapoint, len(ra.Index))
	for i, idx := range ra.Index {
		anoms[i] = Datapoint{Timestamp: int64(idx), Value: ra.Anoms[i]}
	}

	res.PngData = response.Data
	res.Anomalies = anoms
	res.Method = "vec"

	return res, nil
}

func formatResponseTs(response tsResponse) (Result, error) {
	var res Result

	ra := response.Anoms
	if len(ra.Anoms) != len(ra.Timestamp) {
		return res, fmt.Errorf("non-equal number of anomalies and timestamps (%d != %d)", len(ra.Anoms), len(ra.Timestamp))
	}

	anoms := make([]Datapoint, len(ra.Anoms))
	for i, tsf := range ra.Timestamp {
		anoms[i] = Datapoint{Timestamp: int64(tsf), Value: ra.Anoms[i]}
	}

	res.PngData = response.Data
	res.Anomalies = anoms
	res.Method = "ts"

	return res, nil
}

func rCmdString(method, args string) (string, error) {
	switch method {
	case "ts":
		return fmt.Sprintf(`processAnomsTs("%s")`, args), nil
	case "vec":
		return fmt.Sprintf(`processAnomsVec("%s")`, args), nil
	default:
		return "", errors.New("method must be 'ts' or 'vec'")
	}
}

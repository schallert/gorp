package rserve

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"sort"
	"strconv"
	"time"
)

// A Datapoint represents a Unix timestamp and a corresponding float value.
type Datapoint struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

// MarshalJSON encodes a datapoint as an array consisting of (in order) the
// timestamp and value.
func (d Datapoint) MarshalJSON() ([]byte, error) {
	data := [2]interface{}{d.Timestamp, d.Value}
	return json.Marshal(data)
}

// UnmarshalJSON unmarshals an array consisting of (in order) a timestamp and
// a float value. Because datapoints from R/roger are decoded as float64s, we
// check if the timestamp is a float64 and if so convert it to an int64.
func (d *Datapoint) UnmarshalJSON(b []byte) (err error) {
	var data [2]json.Number
	err = json.Unmarshal(b, &data)
	if err != nil {
		return
	}

	d.Timestamp, err = data[0].Int64()
	if err != nil {
		// json-encoded R structs are decoded as float64, so if we fail to parse
		// an int64 we treat it as a float64
		var ts float64
		ts, err2 := data[0].Float64()
		if err2 != nil {
			return err2
		}
		d.Timestamp = int64(ts)
	}

	d.Value, err = data[1].Float64()
	if err != nil {
		return
	}

	return
}

type datapoints []Datapoint

func (dps datapoints) Len() int           { return len(dps) }
func (dps datapoints) Less(i, j int) bool { return dps[i].Timestamp < dps[j].Timestamp }
func (dps datapoints) Swap(i, j int)      { dps[i], dps[j] = dps[j], dps[i] }

func (dps datapoints) writeTempCsv() (string, error) {
	tf, err := ioutil.TempFile("", "gorp-")
	if err != nil {
		return "", fmt.Errorf("tempfile err: %v", err)
	}

	err = dps.writeCsv(tf)
	if err != nil {
		return "", fmt.Errorf("writeCsv err: %v", err)
	}

	// close it since rserve will be reading it
	tf.Close()

	return tf.Name(), nil
}

func (dps datapoints) writeCsv(w io.Writer) (err error) {
	sort.Sort(dps)

	cw := csv.NewWriter(w)
	cw.Write([]string{"date", "value"})
	for _, dp := range dps {
		t := time.Unix(dp.Timestamp, 0)
		ts := t.Format("2006-01-02 15:04:05")
		vs := strconv.FormatFloat(dp.Value, 'f', -1, 64)
		err = cw.Write([]string{ts, vs})
		if err != nil {
			return
		}
	}
	cw.Flush()
	return
}

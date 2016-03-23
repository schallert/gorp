# Gorp

[![GoDoc](https://godoc.org/github.com/schallert/gorp?status.svg)](https://godoc.org/github.com/schallert/gorp) [![Build Status](https://travis-ci.org/schallert/gorp.svg?branch=master)](https://travis-ci.org/schallert/gorp)

Gorp provides an API for sending timeseries datapoints to be processed using Twitter's [AnomalyDetection R package](https://github.com/twitter/AnomalyDetection). It returns the resulting PNG, or a JSON object containing the base64 encoded PNG and additionally an array of anomalous datapoints.

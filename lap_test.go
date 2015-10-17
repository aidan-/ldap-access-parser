package main

import (
	"bytes"
	//"fmt"
	"io/ioutil"
	"strings"
	"testing"
)

func TestTruth(t *testing.T) {
	if true != true {
		t.Error("everything I know is wrong")
	}
}

func TestEventPrintJSON(t *testing.T) {
	var b bytes.Buffer
	format := "json"
	c := config{Output: &b, OutputFormat: &format}

	events := []Event{Event{}}
	eventJSON := []string{`{"time":"","client":"","server":"","connection":0,"ssl":false,"operation":0,"action":"","requests":null,"responses":null}`}

	for n, e := range events {
		c.printEvent(e)
		if strings.TrimRight(b.String(), "\n") != eventJSON[n] {
			t.Errorf("TestEventPrintJSON %d failed: \n\texpected : '%s' \n\tgot: '%s'\n", n, eventJSON[n], b.String())
		}
	}
}

func TestDuration(t *testing.T) {
	data := []struct{ start, stop string }{
		{"21/Apr/2009:11:39:55 -0700", "21/Apr/2010:11:39:55 -0700"},
		{"21/Apr/2010:11:39:55 -0700", "21/Apr/209:11:39:55 -0700"}}
	durations := []int{31536000, -1}

	for n, input := range data {
		res, _ := timeDuration(input.start, input.stop)

		if res != durations[n] {
			t.Errorf("TestDuration %d failed: %d didn't match %d\n", n+1, res, durations[n])
		}
	}

}

func TestParseFileXML(t *testing.T) {
	compileRegexes()
	var b bytes.Buffer
	format := "xml"
	tail := false

	c := config{Output: &b,
		OutputFormat: &format,
		TailFile:     &tail}

	testInput := []string{"test_data/input_1.txt", "test_data/input_2.txt", "test_data/input_3.txt", "test_data/input_4.txt", "test_data/input_5.txt"}
	testOutput := []string{"test_data/output_1_xml.txt", "test_data/output_2_xml.txt", "test_data/output_3_xml.txt", "test_data/output_4_xml.txt", "test_data/output_5_xml.txt"}

	for n, file := range testInput {
		ac := map[int]Event{}
		c.parseFile(ac, file)

		if expected, err := ioutil.ReadFile(testOutput[n]); err == nil {
			if b.String() != string(expected) {
				t.Errorf("TestParseFileXML %d failed: got %s\nexpected: %s\n", n+1, b.String(), string(expected))
			}
		}
		b.Reset()
	}
}

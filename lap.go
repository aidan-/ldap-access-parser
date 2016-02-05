package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/hpcloud/tail"
)

const (
	timeFormat = "02/Jan/2006:03:04:05 -0700"
	version = "v1.0"
)

//Event (request(s) paired with response) to be output in either
//JSON or XML depending on user preference.
type Event struct {
	OppTime         string   `json:"time" xml:"DateTime"`
	Client          string   `json:"client" xml:"Client"`
	Server          string   `json:"server" xml:"Server"`
	Connection      int      `json:"connection" xml:"Connection"`
	SSL             bool     `json:"ssl" xml:"SSL"`
	SSLCipher       string   `json:"sslcipher,omitempty" xml:"SSLCipher,omitempty"`
	SSLStrength     string   `json:"sslstrength,omitempty" xml:"SSLStrength,omitempty"`
	Operation       int      `json:"operation" xml:"Operation"`
	AuthenticatedDN string   `json:"authenticateddn,omitempty" xml:"AuthenticatedDN,omitempty"`
	Action          string   `json:"action" xml:"Action"`
	Requests        []string `json:"requests" xml:"Requests>Request"`
	Responses       []string `json:"responses" xml:"Responses>Response"`
	Duration        int      `json:"duration,omitempty" xml:"Duration,omitempty"`
	ConnTime        string   `json:"-" xml:"-"`
}

type config struct {
	Version      *bool
	TailFile     *bool
	OutputFormat *string
	LogFiles     *[]string
	Output       io.Writer
}

//holds config
var c config

//regexes to extract relevent fields from log lines
var lineMatch = `^\[(?P<time>.*)\] conn=(?P<conn_num>\d+) (?P<event>.*)`
var connectionMatch = `(?P<ssl>SSL)? connection from (?P<client_ip>.*) to (?P<server_ip>.*)`
var operationMatch = `op=(?P<opnum>\-?\d+) (?P<operation>\w+)(?P<details>.+)?`
var bindDNMatch = `dn=\"(?P<dn>.+)\"`
var connectionClosedMatch = ` closed `
var sslCipherMatch = `SSL (?P<strength>.*)-bit (?P<cipher>.*)`

var lineRe *regexp.Regexp
var connectionOpenRe *regexp.Regexp
var operationRe *regexp.Regexp
var bindDNRe *regexp.Regexp
var connectionClosedRe *regexp.Regexp
var sslCipherRe *regexp.Regexp

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func matchLine(re *regexp.Regexp, line string) (map[string]string, bool) {
	matches := re.FindStringSubmatch(line)
	if len(matches) < 1 {
		return map[string]string{}, false
	}

	kvmap := make(map[string]string)
	for n, k := range re.SubexpNames() {
		kvmap[k] = matches[n]
	}

	return kvmap, true
}

func timeDuration(start string, end string) (int, error) {
	startTime, err := time.Parse(timeFormat, start)
	if err != nil {
		return -1, err
	}
	endTime, err := time.Parse(timeFormat, end)
	if err != nil {
		return -1, err
	}

	duration := endTime.Sub(startTime)

	return int(duration / time.Second), nil
}

func init() {
	c.Version = flag.Bool("V", false, "prints version information")
	c.TailFile = flag.Bool("tail", false, "tail the log file to receive future events")
	c.OutputFormat = flag.String("format", "json", "format to output log events.  possible values are 'json' or 'xml'.")
	c.Output = os.Stdout //configurable to help with unit testing

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}
}

func (c config) printEvent(event Event) {
	var output []byte
	var err error
	if format := *c.OutputFormat; format == "xml" {
		output, err = xml.MarshalIndent(event, "", "    ")
	} else {
		output, err = json.Marshal(event)
	}

	if err == nil {
		fmt.Fprintln(c.Output, string(output))
	}
}

func (c config) parseFile(ac map[int]Event, f string) map[int]Event {
	tc := tail.Config{}
	if *c.TailFile {
		tc.Follow = true
		tc.ReOpen = true
	}

	//open file for parsing
	t, err := tail.TailFile(f, tc)
	check(err)

	//loop through file contents
	for line := range t.Lines {
		var lineMap map[string]string
		var ok bool
		if lineMap, ok = matchLine(lineRe, line.Text); !ok {
			continue
		}

		connum, err := strconv.Atoi(lineMap["conn_num"])
		if err != nil {
			fmt.Printf("failed to parse '%s' into int\n", lineMap["conn_num"])
		}

		if connectionMap, ok := matchLine(connectionOpenRe, lineMap["event"]); ok {
			//new connection made
			ssl := false
			if connectionMap["ssl"] == "SSL" {
				ssl = true
			}

			ac[connum] = Event{
				Client:     connectionMap["client_ip"],
				Server:     connectionMap["server_ip"],
				Connection: connum,
				Operation:  -2, //number that shouldn't exist in logs
				ConnTime:   lineMap["time"],
				SSL:        ssl,
			}

			continue
		}

		if _, exists := ac[connum]; !exists {
			//skip operation if no connection exists
			// this is caused by connections that were created before we started
			// parsing the log file.
			continue
		}

		conn := ac[connum]

		if sslMap, ok := matchLine(sslCipherRe, lineMap["event"]); ok {
			conn.SSLCipher = sslMap["cipher"]
			conn.SSLStrength = sslMap["strength"]
			ac[connum] = conn
		}

		if operationMap, ok := matchLine(operationRe, lineMap["event"]); ok {
			//new operation
			opnum, err := strconv.Atoi(operationMap["opnum"])
			if err != nil {
				fmt.Printf("failed to parse '%s' into int\n", operationMap["opnum"])
			}

			if opnum != conn.Operation {
				if conn.Operation != -2 {
					c.printEvent(conn)
				}
				if operationMap["operation"] == "BIND" {
					if bindDN, ok := matchLine(bindDNRe, lineMap["event"]); ok {
						conn.AuthenticatedDN = bindDN["dn"]
					} else {
						conn.AuthenticatedDN = "__anonymous__"
					}
				}

				conn.OppTime = lineMap["time"]
				conn.Operation = opnum
				conn.Action = operationMap["operation"]
				conn.Requests = make([]string, 0)
				conn.Responses = make([]string, 0)
				conn.Requests = append(conn.Requests, operationMap["operation"]+operationMap["details"])
			} else {
				if operationMap["operation"] == "SORT" || operationMap["operation"] == "VLV" {
					conn.Requests = append(conn.Requests, operationMap["operation"]+operationMap["details"])
				} else {
					conn.Responses = append(conn.Responses, operationMap["operation"]+operationMap["details"])

					c.printEvent(conn)
					conn.Operation = -2
				}
			}

			ac[connum] = conn
		}

		if connectionClosedRe.MatchString(lineMap["event"]) {
			delete(ac, connum)
		}
	}
	return ac
}

func compileRegexes() {
	lineRe = regexp.MustCompile(lineMatch)
	connectionOpenRe = regexp.MustCompile(connectionMatch)
	operationRe = regexp.MustCompile(operationMatch)
	bindDNRe = regexp.MustCompile(bindDNMatch)
	connectionClosedRe = regexp.MustCompile(connectionClosedMatch)
	sslCipherRe = regexp.MustCompile(sslCipherMatch)
}

func main() {
	//prepare regex's
	compileRegexes()
	activeConnections := map[int]Event{}

	flag.Parse()

	if *c.Version {
		fmt.Printf("%s %s\n", os.Args[0], version)
		os.Exit(0)
	}

	if len(flag.Args()) < 1 {
		fmt.Println("ERROR: You must specify at least one log file.")
		flag.Usage()
	}

	c.parseFile(activeConnections, flag.Args()[0])

}

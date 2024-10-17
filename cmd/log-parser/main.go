package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"vim-arcade.theprimeagen.com/pkg/assert"
	prettylog "vim-arcade.theprimeagen.com/pkg/pretty-log"
)

type Log struct {
    Process string `json:"process"`
    Area string `json:"area"`
    Msg string `json:"msg"`
}

type LogLine struct {
    Line string
    Log Log
}

type Filter struct {
    process string
    area string
    msg string
}

type RoundFilter struct  {
    currentRound int
    expectedRound int
}

func NewRoundFilter(round int) IFilter {
    return &RoundFilter{
        currentRound: -1,
        expectedRound: round,
    }
}

func (r *RoundFilter) Filter(line LogLine) bool {
    if simRoundFilter.Filter(line) {
        r.currentRound = getRound(line)
    }

    return r.currentRound == r.expectedRound
}

func (r *RoundFilter) String() string {
    return fmt.Sprintf("RoundFilter(%d)", r.expectedRound)
}

type IFilter interface {
    Filter(line LogLine) bool
    String() string
}

func NewFilter(line string) Filter {
    parts := strings.Split(line, ":")
    assert.Assert(len(parts) == 3, "poorly formed filter", "filter", line)

    return Filter{
        process: parts[0],
        area: parts[1],
        msg: parts[2],
    }
}

func (f *Filter) Filter(log LogLine) bool {
    process := f.process == "*" || strings.Contains(log.Log.Process, f.process)
    area := f.area == "*" || strings.Contains(log.Log.Area, f.area)
    msg := f.msg == "*" || strings.Contains(log.Log.Msg, f.msg)

    return process && area && msg
}

var simRoundFilter = NewFilter("*:*:SimRound")
func getRound(log LogLine) int {
    var line map[string]any
    err := json.Unmarshal([]byte(log.Line), &line)
    assert.NoError(err, "unable to parse sim round")

    return int(line["round"].(float64))
}

func (f *Filter) String() string {
    return fmt.Sprintf("%s:%s:%s", f.process, f.area, f.msg)
}

type Parser struct {
    reader *bufio.Scanner
    filters []IFilter
}

func NewParser(fh *os.File, filters []IFilter) Parser {
    return Parser{
        reader: bufio.NewScanner(fh),
        filters: filters,
    }
}

func (p *Parser) Next() *LogLine {
    var log *LogLine = nil

    outer:
    for p.reader.Scan() {
        txt := p.reader.Text()
        l := toLog(txt)

        if len(p.filters) == 0 {
            log = &l
            break
        }

        for _, f := range p.filters {
            if f.Filter(l) {
                log = &l
                break outer
            }
        }
    }

    return log
}

func (p *Parser) String() string {
    out := []string{ }
    for _, f := range p.filters {
        out = append(out, f.String())
    }

    return strings.Join(out, "\n")
}

func toLog(line string) LogLine {
    var log Log
    _ = json.Unmarshal([]byte(line), &log)

    return LogLine{Log: log, Line: line}
}

func toLogs(lines []string) []LogLine {
    out := []LogLine{}
    for _, line := range lines {
        out = append(out, toLog(line))
    }

    return out
}

func toFilters(lines []string) []IFilter {
    out := []IFilter{}
    for _, line := range lines {
        if line == "" {
            continue
        }
        f := NewFilter(line)
        out = append(out, &f)
    }

    return out
}

func p(msg string, count int) {
    if !strings.HasSuffix(msg, "\n") {
        msg += "\n"
    }

    if count > 1 {
        fmt.Printf("%d: %s", count, msg)
    } else {
        fmt.Printf("%s", msg)
    }
}

func isPipedStdin() bool {
    info, err := os.Stdin.Stat()
    assert.NoError(err, "unable to stat stdin")
    return (info.Mode() & os.ModeCharDevice) == 0
}

func main() {
    pretty := false
    flag.BoolVar(&pretty, "pretty", false, "to make the logs pretty")

    dedupe := false
    flag.BoolVar(&dedupe, "dedupe", false, "dedupe exactly the same logs")

    round := -1
    flag.IntVar(&round, "round", -1, "which log round to grab")

    filtersList := ""
    flag.StringVar(&filtersList, "filters", "", "the filters")
    flag.Parse()

    var fh *os.File = nil
    var err error = nil
    if isPipedStdin() {
        fh = os.Stdin
    } else {
		fh, err = os.OpenFile(flag.Arg(0), os.O_RDWR|os.O_CREATE, 0644)
    }

    assert.NoError(err, "expected contents to be read")

    filtersStrings := strings.Split(filtersList, ",")
    filters := toFilters(filtersStrings)

    if round >= 0 {
        filters = append(filters, NewRoundFilter(round))
    }

    parser := NewParser(fh, filters)

    prev := ""
    count := 0
    for {
        out := parser.Next()
        var toPrint string
        var err error = nil

        if out != nil {
            if pretty {
                var line map[string]any
                err := json.Unmarshal([]byte(out.Line), &line)
                if err == nil {
                    toPrint, err = prettylog.PrettyLine(line, prettylog.Colorizer)
                } else {
                    toPrint = out.Line
                }
            } else {
                toPrint = out.Line
            }
        } else {
            break
        }

        if toPrint == prev {
            count += 1
        } else if prev == "" {
            prev = toPrint
            count = 1
        } else {
            p(prev, count)
            prev = toPrint
            count = 1
        }

        assert.NoError(err, "pretty print should not error")
    }

    p(prev, count)
}


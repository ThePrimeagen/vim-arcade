package main

import (
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

type IFilter interface {
    Filter(line string) bool
    Parse(line []string) (string, int)
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
    process := f.process == "*" || f.process == log.Log.Process
    area := f.area == "*" || f.area == log.Log.Area
    msg := f.msg == "*" || strings.Contains(log.Log.Msg, f.msg)

    return process && area && msg
}

func (f *Filter) String() string {
    return fmt.Sprintf("%s:%s:%s", f.process, f.area, f.msg)
}

type Parser struct {
    logs []LogLine
    filters []Filter
}

func (p *Parser) Next() *LogLine {
    idx := -1
    outer:
    for i, l := range p.logs {
        for _, f := range p.filters {

            if f.Filter(l) {
                idx = i
                break outer
            }
        }
    }

    if idx == -1 {
        return nil
    }

    item := p.logs[idx]
    p.logs = p.logs[idx + 1:]
    return &item
}

func (p *Parser) String() string {
    out := []string{
        fmt.Sprintf("Parser(%d)", len(p.logs)),
        "Filters",
    }
    for _, f := range p.filters {
        out = append(out, f.String())
    }

    return strings.Join(out, "\n")
}

func toLogs(lines []string) []LogLine {
    out := []LogLine{}
    for _, line := range lines {
        var log Log
        _ = json.Unmarshal([]byte(line), &log)
        out = append(out, LogLine{Log: log, Line: line})
    }

    return out
}

func toFilters(lines []string) []Filter {
    out := []Filter{}
    for _, line := range lines {
        out = append(out, NewFilter(line))
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

func main() {
    pretty := false
    flag.BoolVar(&pretty, "pretty", false, "to make the logs pretty")

    dedupe := false
    flag.BoolVar(&dedupe, "dedupe", false, "dedupe exactly the same logs")

    itemsList := ""
    flag.StringVar(&itemsList, "items", "", "the filters")
    flag.Parse()
    contents, err := os.ReadFile(flag.Arg(0))
    assert.NoError(err, "expected contents to be read", "contents", contents)

    itemsStrings := strings.Split(itemsList, ",")
    items := toFilters(itemsStrings)

    lines := strings.Split(string(contents), "\n")
    parser := Parser{ logs: toLogs(lines), filters: items}

    prev := ""
    count := 0

    fmt.Println(parser.String())
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


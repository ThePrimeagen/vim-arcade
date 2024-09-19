package prettylog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"

	"vim-arcade.theprimeagen.com/pkg/assert"
)

const (
	timeFormat = "[15:04:05.000]"

	reset = "\033[0m"

	black        = 30
	red          = 31
	green        = 32
	yellow       = 33
	blue         = 34
	magenta      = 35
	cyan         = 36
	lightGray    = 37
	darkGray     = 90
	lightRed     = 91
	lightGreen   = 92
	lightYellow  = 93
	lightBlue    = 94
	lightMagenta = 95
	lightCyan    = 96
	white        = 97
)

const LevelTrace = slog.LevelDebug - 4
const LevelFatal = slog.LevelError + 4
const ProcessKey = "process"
const AreaKey = "area"
var allColors = []int{
	31,
	32,
	33,
	34,
	35,
	36,
	37,
	90,
	91,
	92,
	93,
	94,
	95,
	96,
	97,
}

// TODO make this better
func getProcessColor(process string) int {
    switch process {
    case "sim":
        return lightGreen
    case "DummyServer":
        return lightBlue
    }
    return lightMagenta
}

var areaColors = map[string]int{}
var areaColorsIdx = 0
func getAreaColor(area string) int {
    color, ok := areaColors[area]
    if !ok {
        color = allColors[areaColorsIdx % len(allColors)]
        areaColors[area] = color
        areaColorsIdx++
    }

    return color
}

func stringifyAttrs(attrs map[string]any) string {
    str := strings.Builder{}
    for k, v := range attrs {
        str.WriteString(k)
        str.WriteString("=")

        switch v.(type) {
        // TODO Go deep, go long, and figure out if there is a better way here
        case string,int,float32,float64,int8,int16,int32,int64,uint,uint8,uint16,uint32,uint64:
            str.WriteString(fmt.Sprintf("%v", v))
        default:
            str.WriteString(fmt.Sprintf("%+v", v))
        }
        str.WriteString(" ")
    }
    return str.String()
}


func colorizer(colorCode int, v string) string {
	return fmt.Sprintf("\033[%sm%s%s", strconv.Itoa(colorCode), v, reset)
}

type Handler struct {
	handler          slog.Handler
	r                func([]string, slog.Attr) slog.Attr
	buf              *bytes.Buffer
	mutex            *sync.Mutex
	writer           io.Writer
	timestamp        bool
	colorize         bool
	outputEmptyAttrs bool
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{handler: h.handler.WithAttrs(attrs), buf: h.buf, r: h.r, mutex: h.mutex, writer: h.writer, colorize: h.colorize}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{handler: h.handler.WithGroup(name), buf: h.buf, r: h.r, mutex: h.mutex, writer: h.writer, colorize: h.colorize}
}

func (h *Handler) computeAttrs(
	ctx context.Context,
	r slog.Record,
) (map[string]any, error) {
	h.mutex.Lock()
	defer func() {
		h.buf.Reset()
		h.mutex.Unlock()
	}()
	if err := h.handler.Handle(ctx, r); err != nil {
		return nil, fmt.Errorf("error when calling inner handler's Handle: %w", err)
	}

	var attrs map[string]any
	err := json.Unmarshal(h.buf.Bytes(), &attrs)
	if err != nil {
		return nil, fmt.Errorf("error when unmarshaling inner handler's Handle result: %w", err)
	}
	return attrs, nil
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	colorize := func(code int, value string) string {
		return value
	}

	if h.colorize {
		colorize = colorizer
	}

	var level string
	levelAttr := slog.Attr{
		Key:   slog.LevelKey,
		Value: slog.AnyValue(r.Level),
	}

	if h.r != nil {
		levelAttr = h.r([]string{}, levelAttr)
	}

	if !levelAttr.Equal(slog.Attr{}) {
		level = levelAttr.Value.String() + ":"

		switch r.Level {
		case LevelTrace: fallthrough
		case slog.LevelDebug:
			level = colorize(lightGray, level)
		case slog.LevelInfo:
			level = colorize(cyan, level)
		case slog.LevelWarn:
			level = colorize(lightYellow, level)
		case slog.LevelError:
			level = colorize(red, level)
		case LevelFatal:
			level = colorize(magenta, level)
		default:
			assert.Never("unrecognized log level", "level", r.Level)
		}
	}

    var timestamp string
    timeAttr := slog.Attr{
        Key:   slog.TimeKey,
        Value: slog.StringValue(r.Time.Format(timeFormat)),
    }
    if h.r != nil {
        timeAttr = h.r([]string{}, timeAttr)
    }
    if !timeAttr.Equal(slog.Attr{}) {
        timestamp = colorize(lightGray, timeAttr.Value.String())
    }

	var msg string
	msgAttr := slog.Attr{
		Key:   slog.MessageKey,
		Value: slog.StringValue(r.Message),
	}
	if h.r != nil {
		msgAttr = h.r([]string{}, msgAttr)
	}
	if !msgAttr.Equal(slog.Attr{}) {
		msg = colorize(white, msgAttr.Value.String())
	}

	attrs, err := h.computeAttrs(ctx, r)
	if err != nil {
		return err
	}

    process, ok := attrs[ProcessKey]
    assert.Assert(ok, "must provide process for my delicious pretty log")
    area, ok := attrs[AreaKey]
    assert.Assert(ok, "must provide area for my delicious pretty log")

    delete(attrs, ProcessKey)
    delete(attrs, AreaKey)

	var attrsAsBytes []byte
	if h.outputEmptyAttrs || len(attrs) > 0 {
        attrString := stringifyAttrs(attrs)
        if len(attrString) > 42 {
            attrsAsBytes, err = json.MarshalIndent(attrs, "", "  ")
            if err != nil {
                return fmt.Errorf("error when marshaling attrs: %w", err)
            }
        } else {
            attrsAsBytes = []byte(attrString)
        }
	}

	out := strings.Builder{}
	if h.timestamp && len(timestamp) > 0 {
		out.WriteString(timestamp)
		out.WriteString(" ")
	}

    out.WriteString(colorize(getProcessColor(process.(string)), process.(string)))
    out.WriteString(":")
    out.WriteString(colorize(getAreaColor(area.(string)), area.(string)))
    out.WriteString(" ")

    out.WriteString(level)
    out.WriteString(" ")
    out.WriteString(msg)
    out.WriteString(" ")

	if len(attrsAsBytes) > 0 {
		out.WriteString(colorize(lightGray, string(attrsAsBytes)))
	}

	_, err = io.WriteString(h.writer, out.String()+"\n")
	if err != nil {
		return err
	}

	return nil
}

func suppressDefaults(
	next func([]string, slog.Attr) slog.Attr,
) func([]string, slog.Attr) slog.Attr {
	return func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey ||
			a.Key == slog.LevelKey ||
			a.Key == slog.MessageKey {
			return slog.Attr{}
		}
		if next == nil {
			return a
		}
		return next(groups, a)
	}
}

func New(handlerOptions *slog.HandlerOptions, options ...Option) *Handler {
	if handlerOptions == nil {
		handlerOptions = &slog.HandlerOptions{}
	}

	buf := &bytes.Buffer{}
	handler := &Handler{
		buf: buf,
        timestamp: false,
		handler: slog.NewJSONHandler(buf, &slog.HandlerOptions{
			Level:       handlerOptions.Level,
			AddSource:   handlerOptions.AddSource,
			ReplaceAttr: suppressDefaults(handlerOptions.ReplaceAttr),
		}),
		r:     handlerOptions.ReplaceAttr,
		mutex: &sync.Mutex{},
	}

	for _, opt := range options {
		opt(handler)
	}

	return handler
}

func NewHandler(opts *slog.HandlerOptions, params PrettyLoggerParams, options... Option) *Handler {
    options = append([]Option{
        WithDestinationWriter(params.Out),
        WithColor(),
        WithOutputEmptyAttrs(),
    }, options...)
	return New(opts, options...)
}

type Option func(h *Handler)

func WithTimestamp(writer io.Writer) Option {
	return func(h *Handler) {
		h.timestamp = true
	}
}

func WithDestinationWriter(writer io.Writer) Option {
	return func(h *Handler) {
		h.writer = writer
	}
}

func WithColor() Option {
	return func(h *Handler) {
		h.colorize = true
	}
}

func WithOutputEmptyAttrs() Option {
	return func(h *Handler) {
		h.outputEmptyAttrs = true
	}
}

type PrettyLoggerParams struct {
	Out   *os.File
	Level slog.Level
}

func SetProgramLevelPrettyLogger(params PrettyLoggerParams) *slog.Logger {
	prettyHandler := NewHandler(&slog.HandlerOptions{
		Level:       params.Level,
		AddSource:   false,
		ReplaceAttr: nil,
	}, params)
	logger := slog.New(prettyHandler)
	slog.SetDefault(logger)
	return logger
}

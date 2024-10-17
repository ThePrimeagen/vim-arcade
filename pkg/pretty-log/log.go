package prettylog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"slices"
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
		color = allColors[areaColorsIdx%len(allColors)]
		areaColors[area] = color
		areaColorsIdx++
	}

	return color
}

func isHandledKey(key string) bool {
    return key == ProcessKey || key == AreaKey || key == slog.LevelKey ||
        key == slog.MessageKey || key == slog.TimeKey
}

func stringifyAttrs(attrs map[string]any) string {
	str := strings.Builder{}
	keys := slices.Sorted(maps.Keys(attrs))

	for _, k := range keys {
        if isHandledKey(k) {
            continue
        }

		v := attrs[k]
		str.WriteString(k)
		str.WriteString("=")

		switch v.(type) {
		// TODO Go deep, go long, and figure out if there is a better way here
		case string, int, float32, float64, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			str.WriteString(fmt.Sprintf("%v", v))
		default:
			str.WriteString(fmt.Sprintf("%+v", v))
		}
		str.WriteString(" ")
	}
	return strings.TrimSpace(str.String())
}

func Colorizer(colorCode int, v string) string {
	return fmt.Sprintf("\033[%sm%s%s", strconv.Itoa(colorCode), v, reset)
}

type Handler struct {
	handler          slog.Handler
	replaceAttr      func([]string, slog.Attr) slog.Attr
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
	return &Handler{handler: h.handler.WithAttrs(attrs), buf: h.buf, replaceAttr: h.replaceAttr, mutex: h.mutex, writer: h.writer, colorize: h.colorize}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{handler: h.handler.WithGroup(name), buf: h.buf, replaceAttr: h.replaceAttr, mutex: h.mutex, writer: h.writer, colorize: h.colorize}
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

func PrettyLine(data map[string]any, colorize func(code int, value string) string) (string, error) {

	process, ok := data[ProcessKey]
	assert.Assert(ok, "must provide process for my delicious pretty log")
	area, ok := data[AreaKey]
	assert.Assert(ok, "must provide area for my delicious pretty log")

	level := data["level"].(string)
    if level == "DEBUG-4" {
        level = "TRACE"
    }

	switch level {
	case "TRACE":
		fallthrough
	case "DEBUG":
		level = colorize(lightGray, level)
	case "INFO":
		level = colorize(cyan, level)
	case "WARN":
		level = colorize(lightYellow, level)
	case "ERROR":
		level = colorize(red, level)
	case "FATAL":
		level = colorize(magenta, level)
	default:
		assert.Never("unrecognized log level", "level", level)
	}

    msg := data["msg"].(string)
    msg = colorize(white, msg)

	var attrsAsBytes []byte
	var err error
    attrString := stringifyAttrs(data)
    if len(attrString) > 42 {
        attrsAsBytes, err = json.MarshalIndent(data, "", "  ")
        if err != nil {
            return "", fmt.Errorf("error when marshaling attrs: %w", err)
        }
    } else {
        attrsAsBytes = []byte(attrString)
    }

	header := strings.Builder{}
	body := strings.Builder{}

	header.WriteString(colorize(getProcessColor(process.(string)), process.(string)))
	header.WriteString(":")
	header.WriteString(colorize(getAreaColor(area.(string)), area.(string)))
	header.WriteString(" ")

	body.WriteString(level)
	body.WriteString(" ")
	body.WriteString(msg)

	if len(attrsAsBytes) > 0 {
		body.WriteString(" ")
		body.WriteString(colorize(lightGray, string(attrsAsBytes)))
	}

	// disabled in de615d835b17974b3eda8c846ae51fe008663d83
	// i think there is a bug in ordering...
	// h.dedupedInnerPrint(header.String(), body.String())
    return header.String() + body.String(), nil
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	colorize := func(code int, value string) string {
		return value
	}

	if h.colorize {
		colorize = Colorizer
	}

	toPretty := map[string]any{}

	levelAttr := slog.Attr{
		Key:   slog.LevelKey,
		Value: slog.AnyValue(r.Level),
	}

	if h.replaceAttr != nil {
		levelAttr = h.replaceAttr([]string{}, levelAttr)
	}

	toPretty[slog.LevelKey] = levelAttr.Value.String()

	msgAttr := slog.Attr{
		Key:   slog.MessageKey,
		Value: slog.StringValue(r.Message),
	}

	if h.replaceAttr != nil {
		msgAttr = h.replaceAttr([]string{}, msgAttr)
	}

	attrs, err := h.computeAttrs(ctx, r)
	if err != nil {
		return err
	}

	toPretty[slog.LevelKey] = levelAttr.Value.String()
	toPretty[slog.MessageKey] = msgAttr.Value.String()
    toPretty[ProcessKey] = attrs[ProcessKey]
    toPretty[AreaKey] = attrs[AreaKey]

	for k, v := range attrs {
        toPretty[k] = v
    }

    str, err := PrettyLine(toPretty, colorize)
    if err != nil {
        return err
    }

	io.WriteString(h.writer, str)
	io.WriteString(h.writer, "\n")

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
		buf:       buf,
		timestamp: false,
		handler: slog.NewJSONHandler(buf, &slog.HandlerOptions{
			Level:       handlerOptions.Level,
			AddSource:   handlerOptions.AddSource,
			ReplaceAttr: suppressDefaults(handlerOptions.ReplaceAttr),
		}),
		replaceAttr: handlerOptions.ReplaceAttr,
		mutex:       &sync.Mutex{},
	}

	for _, opt := range options {
		opt(handler)
	}

	return handler
}

func NewHandler(opts *slog.HandlerOptions, params PrettyLoggerParams, options ...Option) *Handler {
	options = append([]Option{
		WithDestinationWriter(params.Out),
		WithColor(),
		WithOutputEmptyAttrs(),
	}, options...)
	return New(opts, options...)
}

type Option func(h *Handler)

func WithTimestamp() Option {
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

func WithoutColor() Option {
	return func(h *Handler) {
		h.colorize = false
	}
}

func WithOutputEmptyAttrs() Option {
	return func(h *Handler) {
		h.outputEmptyAttrs = true
	}
}

type PrettyLoggerParams struct {
	Out   io.Writer
	Level slog.Level
}

func NewParams(out io.Writer) PrettyLoggerParams {
	return PrettyLoggerParams{
		Level: LevelTrace,

		Out: out,
	}
}

func SetProgramLevelPrettyLogger(params PrettyLoggerParams) *slog.Logger {
	if os.Getenv("NO_PRETTY_LOGGER") != "" {
		return slog.Default()
	}

	prettyHandler := NewHandler(&slog.HandlerOptions{
		Level:       params.Level,
		AddSource:   false,
		ReplaceAttr: nil,
	}, params)
	logger := slog.New(prettyHandler)
	slog.SetDefault(logger)
	return logger
}

func CreateLoggerSink() *os.File {
	var f *os.File
	var err error

	debugLog := os.Getenv("DEBUG_LOG")
	if debugLog == "" {
		f = os.Stderr
	} else {
		f, err = os.OpenFile(debugLog, os.O_RDWR|os.O_CREATE, 0644)
		assert.NoError(err, "unable to create temporary file")
	}

	return f
}

func CreateLoggerFromEnv(out *os.File) *slog.Logger {
	if out == nil {
		out = CreateLoggerSink()
	}

	if os.Getenv("DEBUG_TYPE") == "pretty" {
        return SetProgramLevelPrettyLogger(NewParams(out))
	}

    logger := slog.New(slog.NewJSONHandler(out, nil))
    slog.SetDefault(logger)
    return logger
}

func Trace(log *slog.Logger, msg string, data ...any) {
	log.Log(context.Background(), LevelTrace, msg, data...)
}

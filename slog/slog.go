package slog

import (
	"context"
	"log/slog"
	"runtime"
	"strings"

	"github.com/rs/zerolog"
)

type attributes []slog.Attr

func (a attributes) MarshalZerologObject(e *zerolog.Event) {
	for _, attr := range a {
		appendAttr(e, attr)
	}
}

type logEvent struct {
	group        *zerolog.Event
	nxtGroupName string
	next         *logEvent
}

type Handler struct {
	logger    zerolog.Logger
	head      *logEvent
	current   *logEvent
	groupName string
}

func (h *Handler) Clone() *Handler {
	// Create new logEvent instances and copy their values
	newHead := h.copyLogEvent(h.head)
	newCurrent := h.copyLogEvent(h.current)

	// Create a new instance of Handler with copied values
	return &Handler{
		logger:    h.logger,
		head:      newHead,
		current:   newCurrent,
		groupName: h.groupName,
	}
}

func (h *Handler) copyLogEvent(event *logEvent) *logEvent {
	if event == nil {
		return nil
	}

	// Create a new logEvent instance with copied values
	newEvent := &logEvent{
		group:        event.group,
		nxtGroupName: event.nxtGroupName,
	}

	if event.next != nil {
		newEvent.next = h.copyLogEvent(event.next)
	}

	return newEvent
}

func NewHandler(logger zerolog.Logger) *Handler {
	return &Handler{
		logger: logger,
	}
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.logger.GetLevel() <= convertSlogLevel(level)
}

func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	// Map slog level to the zerolog level
	lvl := convertSlogLevel(record.Level)

	// Create an event for the zerolog mapped level from the slog level. If the
	// level is ERROR, PANIC, or FATAL append the stacktrace to the event.
	event := h.logger.WithLevel(lvl)

	// Add the timestamp to the event
	if !record.Time.IsZero() {
		event.Time(zerolog.TimestampFieldName, record.Time)
	}

	// Add caller information to the event
	if record.PC != 0 {
		frame, _ := runtime.CallersFrames([]uintptr{record.PC}).Next()
		if frame.PC != 0 {
			event.Str(zerolog.CallerFieldName, zerolog.CallerMarshalFunc(frame.PC, frame.File, frame.Line))
		}
	}

	// If the current group is not nil, then add al the attributes to the group
	// TODO: check object equality
	if h.head != nil {
		dict := getCorrectEvent(h.head, nil)

		record.Attrs(func(attr slog.Attr) bool {
			// ignore if attribute is empty
			if attr.Equal(slog.Attr{}) {
				return true
			}
			appendAttr(dict, attr)
			return true
		})

		event.Dict(h.groupName, dict).Msg(record.Message)
	} else {
		record.Attrs(func(attr slog.Attr) bool {
			// ignore if attribute is empty
			if attr.Equal(slog.Attr{}) {
				return true
			}

			appendAttr(event, attr)
			return true
		})
		event.Msg(record.Message)
	}
	return nil
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if h.head == nil {
		return &Handler{
			logger: h.logger.With().EmbedObject(attributes(attrs)).Logger(),
		}
	}

	// New handler
	handler := h.Clone()
	handler.current.group = h.current.group.EmbedObject(attributes(attrs))
	// return the handler with copy of logger, logevent and currentLogEvent
	return handler
}

func (h *Handler) WithGroup(name string) slog.Handler {
	// Ignore if the name is empty
	if strings.TrimSpace(name) == "" {
		return h
	}

	// If this is the first group then
	if h.head == nil {
		currentLogEvent := &logEvent{
			group: zerolog.Dict(),
		}
		return &Handler{
			logger:    h.logger,
			head:      currentLogEvent,
			current:   currentLogEvent,
			groupName: name,
		}
	}

	nextEvent := &logEvent{
		group: zerolog.Dict(),
	}

	handler := h.Clone()
	handler.current.nxtGroupName = name

	handler.current.next = nextEvent
	handler.current = nextEvent

	return handler

}

func appendAttr(evt *zerolog.Event, attr slog.Attr) {
	// Depending on the kind we can simply handle the type by called a method on
	// the slog.Value type to get the real value.
	switch attr.Value.Kind() {
	case slog.KindBool:
		evt.Bool(attr.Key, attr.Value.Bool())
	case slog.KindDuration:
		evt.Dur(attr.Key, attr.Value.Duration())
	case slog.KindFloat64:
		evt.Float64(attr.Key, attr.Value.Float64())
	case slog.KindInt64:
		evt.Int64(attr.Key, attr.Value.Int64())
	case slog.KindString:
		evt.Str(attr.Key, attr.Value.String())
	case slog.KindTime:
		evt.Time(attr.Key, attr.Value.Time())
	case slog.KindUint64:
		evt.Uint64(attr.Key, attr.Value.Uint64())
	case slog.KindLogValuer:
		evt.Interface(attr.Key, attr.Value.Resolve())
	case slog.KindGroup:
		evt.Interface(attr.Key, attr.Value.Resolve())
	default:
		evt.Interface(attr.Key, attr.Value.Any())
	}
}

func convertSlogLevel(l slog.Level) zerolog.Level {
	switch {
	case l >= slog.LevelError:
		return zerolog.ErrorLevel
	case l >= slog.LevelWarn:
		return zerolog.WarnLevel
	case l >= slog.LevelInfo:
		return zerolog.InfoLevel
	default:
		return zerolog.DebugLevel
	}
}

// recursively build the dictionary from the logEvent
func getCorrectEvent(event *logEvent, currentEvent *zerolog.Event) *zerolog.Event {
	if currentEvent == nil {
		currentEvent = event.group
	}

	if event.next != nil {
		currentEvent.Dict(event.nxtGroupName, getCorrectEvent(event.next, nil))
	}

	return currentEvent
}

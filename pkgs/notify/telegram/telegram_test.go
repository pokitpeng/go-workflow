package telegram

import (
	"testing"
)

func TestNewListener(t *testing.T) {
	listener := NewListener(12345, "test_hash", "+8612345678900",
		WithSessionPath("test_session.json"),
		WithChannels(123456789, -1001234567890),
		WithBufferSize(50),
	)

	if listener.appID != 12345 {
		t.Errorf("expected appID 12345, got %d", listener.appID)
	}

	if listener.appHash != "test_hash" {
		t.Errorf("expected appHash test_hash, got %s", listener.appHash)
	}

	if listener.phone != "+8612345678900" {
		t.Errorf("expected phone +8612345678900, got %s", listener.phone)
	}

	if listener.sessionPath != "test_session.json" {
		t.Errorf("expected sessionPath test_session.json, got %s", listener.sessionPath)
	}

	if len(listener.channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(listener.channels))
	}

	if !listener.channels[123456789] {
		t.Error("expected channel 123456789 to be in whitelist")
	}

	if !listener.channels[-1001234567890] {
		t.Error("expected channel -1001234567890 to be in whitelist")
	}

	if listener.bufferSize != 50 {
		t.Errorf("expected bufferSize 50, got %d", listener.bufferSize)
	}
}

func TestMessagesChannel(t *testing.T) {
	listener := NewListener(12345, "test_hash", "+8612345678900")

	msgChan := listener.Messages()
	if msgChan == nil {
		t.Error("expected messages channel to be created")
	}
}

func TestDefaultCodeHandler(t *testing.T) {
	handler := DefaultCodeHandler()
	if handler == nil {
		t.Error("expected default code handler to be set")
	}
}

func TestChannelMessage(t *testing.T) {
	msg := &ChannelMessage{
		ChannelID:   123456789,
		ChannelName: "Test Channel",
		MessageID:   100,
		Text:        "Hello World",
		Date:        1640000000,
		SenderID:    987654321,
	}

	if msg.ChannelID != 123456789 {
		t.Errorf("expected ChannelID 123456789, got %d", msg.ChannelID)
	}

	if msg.ChannelName != "Test Channel" {
		t.Errorf("expected ChannelName Test Channel, got %s", msg.ChannelName)
	}

	if msg.Text != "Hello World" {
		t.Errorf("expected Text Hello World, got %s", msg.Text)
	}
}

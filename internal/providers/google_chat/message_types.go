package google_chat

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type Thread struct {
	ThreadKey string `json:"threadKey"`
}

type CardHeader struct {
	Title        string `json:"title"`
	ImageUrl     string `json:"imageUrl"`
	ImageType    string `json:"imageType"`
	ImageAltText string `json:"imageAltText"`
}

type Section struct {
	Collapsible               bool     `json:"collapsible"`
	UncollapsibleWidgetsCount int      `json:"uncollapsibleWidgetsCount"`
	Widgets                   []Widget `json:"widgets"`
}

type Widget interface {
	WidgetType() string
}

type ColumnsWidgetColumnItem struct {
	HorizontalSizeStyle string   `json:"horizontalSizeStyle"`
	HorizontalAlignment string   `json:"horizontalAlignment"`
	VerticalAlignment   string   `json:"verticalAlignment"`
	Widgets             []Widget `json:"widgets"`
}

type ColumnsWidget struct {
	Columns struct {
		ColumnItems []ColumnsWidgetColumnItem `json:"columnItems"`
	} `json:"columns"`
}

type DecoratedTextWidget struct {
	DecoratedText struct {
		Text        *string `json:"text"`
		WrapText    *bool   `json:"wrapText"`
		BottomLabel *string `json:"bottomLabel"`
	} `json:"decoratedText"`
}

type TextParagraphWidget struct {
	TextParagraph struct {
		Text *string `json:"text"`
	} `json:"textParagraph"`
}

func (c ColumnsWidget) WidgetType() string {
	return "Columns"
}

func (d DecoratedTextWidget) WidgetType() string {
	return "DecoratedText"
}

func (t TextParagraphWidget) WidgetType() string {
	return "TextParagraph"
}

func (s *Section) UnmarshalJSON(data []byte) error {
	var raw struct {
		Collapsible               bool              `json:"collapsible"`
		UncollapsibleWidgetsCount int               `json:"uncollapsibleWidgetsCount"`
		Widgets                   []json.RawMessage `json:"widgets"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	s.Collapsible = raw.Collapsible
	s.UncollapsibleWidgetsCount = raw.UncollapsibleWidgetsCount

	for _, rawWidget := range raw.Widgets {
		widget, err := unmarshalWidget(rawWidget)
		if err != nil {
			return err
		}
		s.Widgets = append(s.Widgets, widget)
	}
	return nil
}

func (cwci *ColumnsWidgetColumnItem) UnmarshalJSON(data []byte) error {
	var raw struct {
		HorizontalSizeStyle string            `json:"horizontalSizeStyle"`
		HorizontalAlignment string            `json:"horizontalAlignment"`
		VerticalAlignment   string            `json:"verticalAlignment"`
		Widgets             []json.RawMessage `json:"widgets"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	cwci.HorizontalSizeStyle = raw.HorizontalSizeStyle
	cwci.HorizontalAlignment = raw.HorizontalAlignment
	cwci.VerticalAlignment = raw.VerticalAlignment

	for _, rawWidget := range raw.Widgets {
		widget, err := unmarshalWidget(rawWidget)
		if err != nil {
			return err
		}
		cwci.Widgets = append(cwci.Widgets, widget)
	}
	return nil
}

type Card struct {
	Header   CardHeader `json:"header"`
	Sections []Section  `json:"sections"`
}

type Cards struct {
	CardId string `json:"cardId"`
	Card   Card   `json:"card"`
}

// ComplexChatMessage represents the structure for sending a
// complex message in Google Chat Webhook endpoint.
// https://developers.google.com/chat/api/reference/rest/v1/spaces.messages
type ComplexChatMessage struct {
	Thread      Thread  `json:"thread"`
	ThreadReply bool    `json:"threadReply"`
	Cards       []Cards `json:"cardsV2"`
}

// BasicChatMessage represents the structure for sending a
// Text message in Google Chat Webhook endpoint.
// https://developers.google.com/chat/api/guides/message-formats/basic
type BasicChatMessage struct {
	Text string `json:"text"`
}

type ChatMessage interface {
	ToBuffer() (*bytes.Buffer, error)
}

func (c ComplexChatMessage) ToBuffer() (*bytes.Buffer, error) {
	return msgToBuffer(c)
}

func (b BasicChatMessage) ToBuffer() (*bytes.Buffer, error) {
	return msgToBuffer(b)
}

func msgToBuffer(msg ChatMessage) (*bytes.Buffer, error) {
	out, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(out), nil
}

// Determine the type of each widget and unmarshal accordingly
// This can be done by inspecting the raw JSON of each widget
// and deciding which struct to unmarshal into
func unmarshalWidget(data json.RawMessage) (Widget, error) {
	// First, unmarshal into a generic map to inspect the JSON structure
	var widgetMap map[string]interface{}
	if err := json.Unmarshal(data, &widgetMap); err != nil {
		return nil, err
	}

	// Example logic for determining the widget type
	// This part depends on the unique structure or fields of your widget types
	if _, exists := widgetMap["columns"]; exists {
		var widget *ColumnsWidget
		if err := json.Unmarshal(data, &widget); err != nil {
			return nil, err
		}
		return widget, nil
	} else if _, exists := widgetMap["decoratedText"]; exists {
		var widget *DecoratedTextWidget
		if err := json.Unmarshal(data, &widget); err != nil {
			return nil, err
		}
		return widget, nil
	} else if _, exists := widgetMap["textParagraph"]; exists {
		var widget *TextParagraphWidget
		if err := json.Unmarshal(data, &widget); err != nil {
			return nil, err
		}
		return widget, nil
	}

	// If no known type is identified, return an error
	return nil, fmt.Errorf("unknown widget type")
}

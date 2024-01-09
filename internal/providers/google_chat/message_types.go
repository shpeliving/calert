package google_chat

import (
	"bytes"
	"encoding/json"
)

type Thread struct {
	Name      string `json:"name"`
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
	ColumnItems []ColumnsWidgetColumnItem `json:"columnItems"`
}

type DecoratedTextWidget struct {
	Text     string `json:"text"`
	WrapText bool   `json:"wrapText"`
}

type TextParagraphWidget struct {
	Text string `json:"text"`
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
	Thread      Thread `json:"thread"`
	ThreadReply bool   `json:"threadReply"`
	Cards       Cards  `json:"cardsV2"`
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

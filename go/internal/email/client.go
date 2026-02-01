// Package email はメール送信機能を提供します
package email

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v2"
)

// Client はメール送信クライアント
type Client struct {
	client    *resend.Client
	fromEmail string
}

// NewClient は Client を生成する
func NewClient(apiKey, fromEmail string) *Client {
	return &Client{
		client:    resend.NewClient(apiKey),
		fromEmail: fromEmail,
	}
}

// SendInput はメール送信の入力パラメータ
type SendInput struct {
	To      string
	Subject string
	HTML    string
}

// Send はメールを送信する
func (c *Client) Send(ctx context.Context, input SendInput) error {
	params := &resend.SendEmailRequest{
		From:    c.fromEmail,
		To:      []string{input.To},
		Subject: input.Subject,
		Html:    input.HTML,
	}

	_, err := c.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return fmt.Errorf("メール送信に失敗しました: %w", err)
	}

	return nil
}

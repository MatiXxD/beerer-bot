package telegram

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/require"
)

type telegramAPIClient struct {
	mu                sync.Mutex
	methods           []string
	forms             []url.Values
	failDeleteWebhook bool
}

func (c *telegramAPIClient) Do(r *http.Request) (*http.Response, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	method := pathParts[len(pathParts)-1]
	form := make(url.Values, len(r.Form))
	for key, values := range r.Form {
		form[key] = append([]string(nil), values...)
	}
	c.mu.Lock()
	c.methods = append(c.methods, method)
	c.forms = append(c.forms, form)
	c.mu.Unlock()

	body := `{"ok":true,"result":[]}`
	switch {
	case method == "deleteWebhook" && c.failDeleteWebhook:
		body = `{"ok":false,"error_code":400,"description":"delete failed"}`
	case method == "getMe":
		body = `{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"test"}}`
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}

func (c *telegramAPIClient) bot(t *testing.T) *tgbotapi.BotAPI {
	t.Helper()
	bot, err := tgbotapi.NewBotAPIWithClient("test", "http://telegram.test/bot%s/%s", c)
	require.NoError(t, err)
	return bot
}

func (c *telegramAPIClient) request(method string) (url.Values, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, got := range c.methods {
		if got == method {
			return c.forms[i], true
		}
	}
	return nil, false
}

func TestNewLongPollingUsesDefaults(t *testing.T) {
	cfg := testConfig()
	api := &telegramAPIClient{}
	lp := NewLongPolling(api.bot(t), cfg.LongPolling)

	require.Equal(t, cfg.LongPolling, lp.cfg)
}

func TestLongPollingUpdatesAndStop(t *testing.T) {
	cfg := testConfig()
	api := &telegramAPIClient{}
	lp := NewLongPolling(api.bot(t), cfg.LongPolling)

	updates, err := lp.Updates()
	require.NoError(t, err)
	require.NotNil(t, updates)
	_, ok := api.request("deleteWebhook")
	require.True(t, ok, "Updates() did not delete the webhook")

	deadline := time.Now().Add(time.Second)
	for {
		if form, ok := api.request("getUpdates"); ok {
			require.Equal(t, "17", form.Get("timeout"))
			break
		}
		if time.Now().After(deadline) {
			require.FailNow(t, "GetUpdatesChan did not request updates")
		}
		time.Sleep(time.Millisecond)
	}

	require.NoError(t, lp.Stop())
	select {
	case _, ok := <-updates:
		require.False(t, ok, "updates channel remained open after Stop()")
	case <-time.After(time.Second):
		require.FailNow(t, "updates channel was not closed after Stop()")
	}
}

func TestLongPollingReturnsDeleteWebhookError(t *testing.T) {
	cfg := testConfig()
	api := &telegramAPIClient{failDeleteWebhook: true}
	lp := NewLongPolling(api.bot(t), cfg.LongPolling)

	updates, err := lp.Updates()
	require.Error(t, err)
	require.Nil(t, updates)
}

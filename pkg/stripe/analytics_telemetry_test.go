package stripe

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// Context Tests
func TestEventMetadataWithGet(t *testing.T) {
	ctx := context.Background()
	event := &CLIAnalyticsEventMetadata{
		InvocationID: "hello",
		UserAgent:    "uesr",
		CLIVersion:   "1.0",
		OS:           "os",
	}
	newCtx := WithEventMetadata(ctx, event)

	require.Equal(t, GetEventMetadata(newCtx), event)
}

func TestGetEventMetadata_DoesNotExistInCtx(t *testing.T) {
	ctx := context.Background()
	require.Nil(t, GetEventMetadata(ctx))
}

func TestTelemetryClientWithGet(t *testing.T) {
	ctx := context.Background()
	url, _ := url.Parse("http://hello.com")
	telemetryClient := &AnalyticsTelemetryClient{
		BaseURL:    url,
		HTTPClient: &http.Client{},
	}
	newCtx := WithTelemetryClient(ctx, telemetryClient)

	require.Equal(t, GetTelemetryClient(newCtx), telemetryClient)
}

func TestGetTelemetryClient_DoesNotExistInCtx(t *testing.T) {
	ctx := context.Background()
	require.Nil(t, GetTelemetryClient(ctx))
}

func TestSetCobraCommandContext(t *testing.T) {
	tel := NewEventMetadata()
	cmd := &cobra.Command{
		Use: "foo",
	}
	tel.SetCobraCommandContext(cmd)
	require.Equal(t, "foo", tel.CommandPath)
}

func TestSetMerchant(t *testing.T) {
	tel := NewEventMetadata()
	merchant := "acct_zzzzzz"
	tel.SetMerchant(merchant)
	require.Equal(t, merchant, tel.Merchant)
}

// AnalyticsClient Tests
func TestSendAPIRequestEvent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		bodyString := string(body)
		require.Contains(t, bodyString, "cli_version=master")
		require.Contains(t, bodyString, "client_id=stripe-cli")
		require.Contains(t, bodyString, "command_path=stripe+test")
		require.Contains(t, bodyString, "event_name=API+Request")
		require.Contains(t, bodyString, "generated_resource=false")
		require.Contains(t, bodyString, "invocation_id=123456")
		require.Contains(t, bodyString, "livemode=false")
		require.Contains(t, bodyString, "merchant=acct_1234")
		require.Contains(t, bodyString, "os=darwin")
		require.Contains(t, bodyString, "request_id=req_zzz")
		require.Contains(t, bodyString, "user_agent=Unit+Test")
	}))
	defer ts.Close()
	baseURL, _ := url.Parse(ts.URL)

	telemetryMetadata := &CLIAnalyticsEventMetadata{
		InvocationID:      "123456",
		UserAgent:         "Unit Test",
		CLIVersion:        "master",
		OS:                "darwin",
		CommandPath:       "stripe test",
		Merchant:          "acct_1234",
		GeneratedResource: false,
	}
	processCtx := WithEventMetadata(context.Background(), telemetryMetadata)
	analyticsClient := AnalyticsTelemetryClient{BaseURL: baseURL, HTTPClient: &http.Client{}}
	resp, err := analyticsClient.SendAPIRequestEvent(processCtx, "req_zzz", false)
	require.NoError(t, err)
	require.NotNil(t, resp)
	resp.Body.Close()
}

func TestSkipsSendAPIRequestEventWhenMetadataIsEmpty(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// do nothing
	}))
	defer ts.Close()
	baseURL, _ := url.Parse(ts.URL)
	analyticsClient := AnalyticsTelemetryClient{BaseURL: baseURL, HTTPClient: &http.Client{}}
	resp, err := analyticsClient.SendAPIRequestEvent(context.Background(), "req_zzz", false)
	require.NoError(t, err)
	require.Nil(t, resp)

	// We shouldn't get here but the linter is unhappy
	if resp != nil {
		resp.Body.Close()
	}
}

func TestSendEvent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		bodyString := string(body)
		require.Contains(t, bodyString, "cli_version=master")
		require.Contains(t, bodyString, "client_id=stripe-cli")
		require.Contains(t, bodyString, "command_path=stripe+test")
		require.Contains(t, bodyString, "event_name=foo")
		require.Contains(t, bodyString, "event_value=bar")
		require.Contains(t, bodyString, "generated_resource=false")
		require.Contains(t, bodyString, "invocation_id=123456")
		require.Contains(t, bodyString, "merchant=acct_1234")
		require.Contains(t, bodyString, "os=darwin")
		require.Contains(t, bodyString, "user_agent=Unit+Test")
	}))
	defer ts.Close()
	baseURL, _ := url.Parse(ts.URL)

	telemetryMetadata := &CLIAnalyticsEventMetadata{
		InvocationID:      "123456",
		UserAgent:         "Unit Test",
		CLIVersion:        "master",
		OS:                "darwin",
		CommandPath:       "stripe test",
		Merchant:          "acct_1234",
		GeneratedResource: false,
	}
	processCtx := WithEventMetadata(context.Background(), telemetryMetadata)
	analyticsClient := AnalyticsTelemetryClient{BaseURL: baseURL, HTTPClient: &http.Client{}}
	analyticsClient.SendEvent(processCtx, "foo", "bar")
}

func TestSkipsSendEventWhenMetadataIsEmpty(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Fail(t, "Did not expect to reach sendData")
		// do nothing
	}))
	defer ts.Close()
	baseURL, _ := url.Parse(ts.URL)

	analyticsClient := AnalyticsTelemetryClient{BaseURL: baseURL, HTTPClient: &http.Client{}}
	analyticsClient.SendEvent(context.Background(), "foo", "bar")
}

// Utility function
func TestTelemetryOptedOut(t *testing.T) {
	require.False(t, TelemetryOptedOut(""))
	require.False(t, TelemetryOptedOut("0"))
	require.False(t, TelemetryOptedOut("false"))
	require.False(t, TelemetryOptedOut("False"))
	require.False(t, TelemetryOptedOut("FALSE"))
	require.True(t, TelemetryOptedOut("1"))
	require.True(t, TelemetryOptedOut("true"))
	require.True(t, TelemetryOptedOut("True"))
	require.True(t, TelemetryOptedOut("TRUE"))
}

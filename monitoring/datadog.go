package monitoring

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
)

type DatadogClient struct {
	Command     string
	Environment string
	ConfigData  map[string]string
	Project     string
	StatusCode  int
	Version     string
}

func NewDatadogClient() *DatadogClient {
	return &DatadogClient{}
}

func (m *DatadogClient) SubmitEvent(panicErr any) {
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to get hostname: %v", err)
	}

	body := datadogV1.EventCreateRequest{
		Title:          fmt.Sprintf("'%s' command run in '%s' project", m.Command, m.Project),
		Text:           m.createEventBody(panicErr),
		AggregationKey: datadog.PtrString("cdflow2"),
		DateHappened:   datadog.PtrInt64(time.Now().Unix()),
		Host:           datadog.PtrString(hostname),
		Tags:           m.collectTags(),
	}

	ctx := context.WithValue(
		context.Background(),
		datadog.ContextAPIKeys,
		map[string]datadog.APIKey{
			"apiKeyAuth": {
				Key: os.Getenv("DD_CLIENT_API_KEY"),
			},
		},
	)

	configuration := datadog.NewConfiguration()
	apiClient := datadog.NewAPIClient(configuration)
	api := datadogV1.NewEventsApi(apiClient)
	_, r, err := api.CreateEvent(ctx, body)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling Datadog `EventsApi.CreateEvent`: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	} else {
		fmt.Fprintf(os.Stderr, "Datadog event submitted.\n")
	}
}

func (m *DatadogClient) collectTags() []string {
	tags := []string{
		"command:" + m.Command,
		"version:" + m.Version,
		"status_code:" + strconv.Itoa(m.StatusCode),
	}

	if m.StatusCode == 0 {
		tags = append(tags, "status:successful")
	} else {
		tags = append(tags, "status:failed")
	}

	if m.Project != "" {
		tags = append(tags, "project:"+m.Project)
	}

	if m.Environment != "" {
		tags = append(tags, "env:"+m.Environment)
	}

	for k, v := range m.ConfigData {
		tags = append(tags, fmt.Sprintf("%s:%s", k, v))
	}

	return tags
}

func (m *DatadogClient) createEventBody(panicErr any) string {
	status := "was successful"
	if panicErr != nil {
		status = "panicked"
	} else if m.StatusCode != 0 {
		status = "failed"
	}

	body := fmt.Sprintf("cdflow2 %s command %s.", m.Command, status)

	if panicErr != nil {
		body += fmt.Sprintf("\nPanic reason:\n%+v", panicErr)
	}

	return body
}

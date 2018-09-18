package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/honeycombio/libhoney-go"
)

// This example reads JSON blobs from a file and sends them as Honeycomb events.
// It will log the success or failure of each sent event to STDOUT.
// The expectation of the file is that it has one JSON blob per line. Each
// JSON blob must have a field named timestamp that includes a time objected
// formatted like RFC3339. We will use that field to set the timestamp for the
// event.

const (
	version           = "0.0.1-example"
	honeyFakeWritekey = "abcabc123123defdef456456"
	honeyDataset      = "example json blobs"
)

var jsonFilePaths = []string{"./example1.json", "./example2.json"}

func main() {

	// basic initialization
	libhConf := libhoney.Config{
		WriteKey: honeyFakeWritekey,
		Dataset:  honeyDataset,
	}
	libhoney.Init(libhConf)
	defer libhoney.Close()
	go readResponses(libhoney.Responses())

	// We want every event to include the number of currently running goroutines
	// and the version number of this app. The goroutines is contrived for this
	// example, but is useful in larger apps. Adding the version number to the
	// global scope means every event sent will include this field.
	libhoney.AddDynamicField("num_goroutines",
		func() interface{} { return runtime.NumGoroutine() })
	libhoney.AddField("read_json_log_version", version)

	// go through each json file and parse it.
	for _, fileName := range jsonFilePaths {
		fh, err := os.Open(fileName)
		if err != nil {
			fmt.Println("Error opening file:", err)
			os.Exit(1)
		}
		defer fh.Close()

		// Create a new builder to store the information about the file being
		// processed. This builder will be passed in to processLine so all events
		// created within that function will have the information about the file
		// and line being processed.
		perFileBulider := libhoney.NewBuilder()
		perFileBulider.AddField("json_file_name", fileName)

		scanner := bufio.NewScanner(fh)
		i := 1
		for scanner.Scan() {
			// each time this is added to the builder the field is overwritten
			perFileBulider.AddField("json_line_number", i)
			i += 1
			processLine(scanner.Text(), perFileBulider)
		}
	}

	fmt.Println("All done! Go check Honeycomb https://ui.honeycomb.io/ to see your data.")
}

func readResponses(responses chan libhoney.Response) {
	for r := range responses {
		if r.StatusCode >= 200 && r.StatusCode < 300 {
			id := r.Metadata.(string)
			fmt.Printf("Successfully sent event %s to Honeycomb\n", id)
		} else {
			fmt.Printf("Error sending event to Honeycomb! Code %d, err %v and response body %s",
				r.StatusCode, r.Err, r.Body)
		}
	}
}

func processLine(line string, builder *libhoney.Builder) {

	// Create the event that this line will fill. because we're creating it from
	// the Builder, it will already have a field containing the name and line
	// number of the JSON file we're parsing.
	ev := builder.NewEvent()
	ev.Metadata = fmt.Sprintf("id %d", rand.Intn(20))
	defer ev.Send()
	defer fmt.Printf("Sending event %s\n", ev.Metadata)

	// unmarshal the JSON blob
	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(line), &data)
	if err != nil {
		ev.AddField("error", err)
		return
	}

	// Override the event timestamp if the JSON blob has a valid time. If time
	// is missing or it doesn't parse correctly, the event will be sent with the
	// default time (Now())
	if timeVal, ok := data["timestamp"]; ok {
		ts, err := time.Parse(time.RFC3339Nano, timeVal.(string))
		if err == nil {
			// we got a valid timestamp. Override the event's timestamp and remove the
			// field from data so we don't get it reported twice
			ev.Timestamp = ts
			delete(data, "timestamp")
		} else {
			ev.AddField("timestamp problem", fmt.Sprintf("problem parsing:%s", err))
		}
	} else {
		ev.AddField("timestamp problem", "missing timestamp")
	}

	// Add all the fields in the JSON blob to the event
	ev.Add(data)

	// Sending is handled by the defer.
}

package v3_helpers

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

type DestinationProcess struct {
	Type string `json:"type"`
}

type Destination struct {
	GUID        string `json:"guid,omitempty"`
	App         App    `json:"app"`
	HTTPVersion int    `json:"http_version,omitempty"`
	Port        int    `json:"port,omitempty"`
	Weight      int    `json:"weight,omitempty"`
}

type Destinations struct {
	Destinations []Destination `json:"destinations"`
}

func InsertDestinations(routeGUID string, destinations []Destination) []string {
	destinationsJSON, err := json.Marshal(Destinations{Destinations: destinations})
	Expect(err).ToNot(HaveOccurred())

	session := cf.Cf("curl", "-f",
		fmt.Sprintf("/v3/routes/%s/destinations", routeGUID),
		"-X", "POST", "-d", string(destinationsJSON))

	Expect(session.Wait()).To(Exit(0))
	response := session.Out.Contents()

	var responseDestinations Destinations
	err = json.Unmarshal(response, &responseDestinations)
	Expect(err).ToNot(HaveOccurred())

	listDstGUIDs := make([]string, 0, len(responseDestinations.Destinations))
	for _, dst := range responseDestinations.Destinations {
		listDstGUIDs = append(listDstGUIDs, dst.GUID)
	}
	return listDstGUIDs
}

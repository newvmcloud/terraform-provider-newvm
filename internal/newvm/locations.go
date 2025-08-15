package newvm

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type NewVmLocationsWrapper struct {
	Locations []Location `json:"locations"`
}

// GetLocations - Returns all operating systems
func (c *Client) GetLocations() ([]Location, error) {
	locations := []Location{}
	// obtain operating systems
	reqLocations, err := http.NewRequest("GET", fmt.Sprintf("%s/backend/com.newvm.network/v1/location", c.HostURL), nil)
	if err != nil {
		return nil, err
	}
	bodyLocations, err := c.doRequest(reqLocations)
	if err != nil {
		return nil, err
	}
	locationsWrapper := NewVmLocationsWrapper{}
	err = json.Unmarshal(bodyLocations, &locationsWrapper)
	if err != nil {
		return nil, err
	}

	// Map response body to model
	for _, record := range locationsWrapper.Locations {
		if record.Provisionable == 1 {
			location := Location{
				ID:         record.ID,
				Name:       record.Name,
				Code:       record.Code,
				ProductIds: record.ProductIds,
			}

			locations = append(locations, location)
		}
	}

	return locations, nil
}

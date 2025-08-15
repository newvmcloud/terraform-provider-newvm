package newvm

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type NewVmOperatingSystemsWrapper struct {
	OperatingSystems []OperatingSystem `json:"result"`
}

// GetOperatingSystems - Returns all operating systems
func (c *Client) GetOperatingSystems() ([]OperatingSystem, error) {
	operatingSystems := []OperatingSystem{}
	// obtain operating systems
	reqOs, err := http.NewRequest("GET", fmt.Sprintf("%s/account/v1/provisioning/os", c.HostURL), nil)
	if err != nil {
		return nil, err
	}
	bodyOs, err := c.doRequest(reqOs)
	if err != nil {
		return nil, err
	}
	osWrapper := NewVmOperatingSystemsWrapper{}
	err = json.Unmarshal(bodyOs, &osWrapper)
	if err != nil {
		return nil, err
	}
	// Map response body to model
	for _, os := range osWrapper.OperatingSystems {
		operatingSystem := OperatingSystem{
			ID:       os.ID,
			Tag:      os.Tag,
			Name:     os.Name,
			Platform: os.Platform,
		}

		operatingSystems = append(operatingSystems, operatingSystem)
	}

	return operatingSystems, nil
}

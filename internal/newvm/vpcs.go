package newvm

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type NewVmVpcsWrapper struct {
	Vpcs []Vpc `json:"vxlan"`
}

type NewVmVpcMembersWrapper struct {
	Members []VpcMember `json:"members"`
}

// GetVpcs - Returns all VPCs
func (c *Client) GetVpcs() ([]Vpc, error) {
	vpcs := []Vpc{}
	// obtain VPCs
	reqVpcs, err := http.NewRequest("GET", fmt.Sprintf("%s/backend/com.newvm.network/v1/vxlan", c.HostURL), nil)
	if err != nil {
		return nil, err
	}
	bodyVpcs, err := c.doRequest(reqVpcs)
	if err != nil {
		return nil, err
	}
	vpcsWrapper := NewVmVpcsWrapper{}
	err = json.Unmarshal(bodyVpcs, &vpcsWrapper)
	if err != nil {
		return nil, err
	}

	// Map response body to model
	for _, record := range vpcsWrapper.Vpcs {
		vpc := Vpc{
			ID:     record.ID,
			Number: record.Number,
			Name:   record.Name,
		}

		vpcs = append(vpcs, vpc)
	}

	return vpcs, nil
}

// GetVpcMembers - Returns all VPC members
func (c *Client) GetVpcMembers() ([]VpcMember, error) {
	vpcMembers := []VpcMember{}
	// obtain VPC members
	reqVpcs, err := http.NewRequest("GET", fmt.Sprintf("%s/backend/com.newvm.network/v1/vxlan/member", c.HostURL), nil)
	if err != nil {
		return nil, err
	}
	bodyVpcMembers, err := c.doRequest(reqVpcs)
	if err != nil {
		return nil, err
	}
	vpcMembersWrapper := NewVmVpcMembersWrapper{}
	err = json.Unmarshal(bodyVpcMembers, &vpcMembersWrapper)
	if err != nil {
		return nil, err
	}

	// Map response body to model
	for _, record := range vpcMembersWrapper.Members {
		vpcMember := VpcMember{
			ID:         record.ID,
			OrderId:    record.OrderId,
			MacAddress: record.MacAddress,
			Vxlan:      record.Vxlan,
		}

		vpcMembers = append(vpcMembers, vpcMember)
	}

	return vpcMembers, nil
}

package newvm

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
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
			ID:        record.ID,
			Number:    record.Number,
			Name:      record.Name,
			Removable: record.Removable,
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
			OrderID:    record.OrderID,
			MacAddress: record.MacAddress,
			Vxlan:      record.Vxlan,
		}

		vpcMembers = append(vpcMembers, vpcMember)
	}

	return vpcMembers, nil
}

// GetVpc - Returns specific VPC details
func (c *Client) GetVpc(ID string) (*Vpc, error) {
	type NewVmVpcWrapper struct {
		Vpc Vpc `json:"vxlan"`
	}

	vpc := Vpc{}
	if ID != "" {
		// obtain all VPCs
		reqVpc, err := http.NewRequest("GET", fmt.Sprintf("%s/backend/com.newvm.network/v1/vxlan/%s", c.HostURL, ID), nil)
		if err != nil {
			return nil, err
		}
		bodyVpc, err := c.doRequest(reqVpc)
		if err != nil {
			return nil, err
		}
		vpcWrapper := NewVmVpcWrapper{}
		err = json.Unmarshal(bodyVpc, &vpcWrapper)
		if err != nil {
			return nil, err
		}

		// Map response body to model
		vpc = vpcWrapper.Vpc
	}
	// log.Printf("VPC: %+v", vpc)
	return &vpc, nil
}

// CreateVpc - Create new VPC order
func (c *Client) CreateVpc(vpc Vpc) (*Vpc, error) {
	// Order @NewVPC Order structure
	type NewVpcOrder struct {
		Name string `json:"label"`
	}

	newVpcOrder := NewVpcOrder{
		Name: vpc.Name,
	}

	rb, err := json.Marshal(newVpcOrder)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/backend/com.newvm.network/v1/vxlan", c.HostURL), strings.NewReader(string(rb)))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	type Result struct {
		ID string `json:"id"`
	}
	var responseBody Result
	err = json.Unmarshal(body, &responseBody)
	if err != nil {
		return nil, err
	}

	vpc.ID = responseBody.ID
	return &vpc, nil
}

// UpdateVpc - Update an existing VPC
func (c *Client) UpdateVpc(ID string, vpc Vpc) error {
	type UpdateVpcOrder struct {
		Name string `json:"label"`
	}

	updateVpcOrder := UpdateVpcOrder{
		Name: vpc.Name,
	}

	rb, err := json.Marshal(updateVpcOrder)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/backend/com.newvm.network/v1/vxlan/%s", c.HostURL, ID), strings.NewReader(string(rb)))
	if err != nil {
		return err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return err
	}

	if strings.ReplaceAll(string(body), " ", "") != "{\"success\":true}" {
		return errors.New(string(body))
	}

	return nil

}

// DeleteVpc - Deletes a VPC
func (c *Client) DeleteVpc(ID string) error {
	reqOrderEnd, err := http.NewRequest("DELETE", fmt.Sprintf("%s/backend/com.newvm.network/v1/vxlan/%s", c.HostURL, ID), nil)
	if err != nil {
		return err
	}
	resBodyOrderEnd, err := c.doRequest(reqOrderEnd)
	if err != nil {
		return err
	}

	if strings.ReplaceAll(string(resBodyOrderEnd), " ", "") != "{\"success\":true}" {
		return errors.New(string(resBodyOrderEnd))
	}

	return nil
}

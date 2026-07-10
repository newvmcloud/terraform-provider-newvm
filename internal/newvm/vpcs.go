package newvm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type NewVmVpcsWrapper struct {
	Vpcs []Vpc `json:"vxlan"`
}

type NewVmVpcMembersWrapper struct {
	Members []VpcMember `json:"members"`
}

const (
	vpcDeleteMaxAttempts = 6
	vpcDeleteRetryDelay  = 5 * time.Second
)

// GetVpcs - Returns all VPCs
func (c *Client) GetVpcs(ctx context.Context) ([]Vpc, error) {
	vpcs := []Vpc{}
	// obtain VPCs
	reqVpcs, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/backend/com.newvm.network/v1/vxlan", c.HostURL), nil)
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
func (c *Client) GetVpcMembers(ctx context.Context) ([]VpcMember, error) {
	vpcMembers := []VpcMember{}
	// obtain VPC members
	reqVpcs, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/backend/com.newvm.network/v1/vxlan/member", c.HostURL), nil)
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
func (c *Client) GetVpc(ctx context.Context, ID string) (*Vpc, error) {
	type NewVmVpcWrapper struct {
		Vpc Vpc `json:"vxlan"`
	}

	vpc := Vpc{}
	if ID != "" {
		// obtain all VPCs
		reqVpc, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/backend/com.newvm.network/v1/vxlan/%s", c.HostURL, ID), nil)
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
func (c *Client) CreateVpc(ctx context.Context, vpc Vpc) (*Vpc, error) {
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

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/backend/com.newvm.network/v1/vxlan", c.HostURL), strings.NewReader(string(rb)))
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
func (c *Client) UpdateVpc(ctx context.Context, ID string, vpc Vpc) error {
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

	req, err := http.NewRequestWithContext(ctx, "PUT", fmt.Sprintf("%s/backend/com.newvm.network/v1/vxlan/%s", c.HostURL, ID), strings.NewReader(string(rb)))
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

func (c *Client) deleteVpcMember(ctx context.Context, vpcID string, orderID int) error {
	type VpcMemberRequest struct {
		OrderID int `json:"orderid"`
	}

	rb, err := json.Marshal(VpcMemberRequest{OrderID: orderID})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "DELETE", fmt.Sprintf("%s/backend/com.newvm.network/v1/vxlan/%s/members", c.HostURL, vpcID), strings.NewReader(string(rb)))
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	return err
}

func (c *Client) getVpcMembersByNumber(ctx context.Context, vpcNumber int32) ([]VpcMember, error) {
	vpcMembers, err := c.GetVpcMembers(ctx)
	if err != nil {
		return nil, err
	}

	filteredMembers := make([]VpcMember, 0, len(vpcMembers))
	for _, member := range vpcMembers {
		if member.Vxlan == vpcNumber {
			filteredMembers = append(filteredMembers, member)
		}
	}

	return filteredMembers, nil
}

func (c *Client) waitForVpcMembersDeletion(ctx context.Context, vpcNumber int32) error {
	var lastErr error

	for attempt := 0; attempt < vpcDeleteMaxAttempts; attempt++ {
		vpcMembers, err := c.getVpcMembersByNumber(ctx, vpcNumber)
		if err == nil && len(vpcMembers) == 0 {
			return nil
		}

		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("VPC %d still has %d members attached", vpcNumber, len(vpcMembers))
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(vpcDeleteRetryDelay):
		}
	}

	return lastErr
}

// DeleteVpc - Deletes a VPC
func (c *Client) DeleteVpc(ctx context.Context, ID string) error {
	vpc, err := c.GetVpc(ctx, ID)
	if err != nil {
		return err
	}

	vpcMembers, err := c.getVpcMembersByNumber(ctx, vpc.Number)
	if err != nil {
		return err
	}

	for _, member := range vpcMembers {
		if err := c.deleteVpcMember(ctx, vpc.ID, member.OrderID); err != nil {
			return err
		}
	}

	if len(vpcMembers) > 0 {
		if err := c.waitForVpcMembersDeletion(ctx, vpc.Number); err != nil {
			return err
		}
	}

	var lastErr error
	for attempt := 0; attempt < vpcDeleteMaxAttempts; attempt++ {
		reqOrderEnd, err := http.NewRequestWithContext(ctx, "DELETE", fmt.Sprintf("%s/backend/com.newvm.network/v1/vxlan/%s", c.HostURL, ID), nil)
		if err != nil {
			return err
		}

		resBodyOrderEnd, err := c.doRequest(reqOrderEnd)
		if err == nil {
			if strings.ReplaceAll(string(resBodyOrderEnd), " ", "") != "{\"success\":true}" {
				return errors.New(string(resBodyOrderEnd))
			}

			return nil
		}

		lastErr = err
		if !strings.Contains(err.Error(), "Failed to trigger VXLAN change") || attempt == vpcDeleteMaxAttempts-1 {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(vpcDeleteRetryDelay):
		}
	}

	return lastErr
}

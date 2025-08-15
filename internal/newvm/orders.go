package newvm

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
)

// GetAllOrders - Returns all user's order
func (c *Client) GetAllOrders() (*[]Order, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/orders", c.HostURL), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	orders := []Order{}
	err = json.Unmarshal(body, &orders)
	if err != nil {
		return nil, err
	}

	return &orders, nil
}

// GetOrder - Returns a specifc order
func (c *Client) GetOrder(orderID string) (*Order, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/account/v1/customer/self/order/%s", c.HostURL, orderID), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	order := Order{}
	err = json.Unmarshal(body, &order)
	if err != nil {
		return nil, err
	}

	return &order, nil
}

func RandomString(n int) string {
	var letters = []rune("0123456789abcdef")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

// CreateOrder - Create new orr
func (c *Client) CreateOrder(orderItems []OrderItem, comments string) (*Order, error) {
	// Order @NewVM Order structure
	type NewVmOrderOption struct {
		VmCore      int `json:"vm_core,omitempty"`
		VmDiskspace int `json:"vm_diskspace,omitempty"`
		VmMem       int `json:"vm_mem,omitempty"`
		VmType      int `json:"vm_type"`
	}
	type NewVmProvisioning struct {
		Hostname    string `json:"hostname,omitempty"`
		Os          string `json:"os,omitempty"`
		VmLocations string `json:"vm_locations,omitempty"`
	}
	type NewVmOrder struct {
		Amount            NewVmOrderOption  `json:"amount,omitempty"`
		CustomDescription string            `json:"custom_description,omitempty"`
		Parent            string            `json:"parent,omitempty"`
		Product           string            `json:"product,omitempty"`
		Provisioning      NewVmProvisioning `json:"provisioning,omitempty"`
		Reference         string            `json:"reference"`
	}
	type NewVmMultiOrder struct {
		Comments   string       `json:"comments,omitempty"`
		PromoCodes []string     `json:"promoCodes,omitempty"`
		Orders     []NewVmOrder `json:"orders,omitempty"`
	}
	multiOrder := NewVmMultiOrder{}
	if comments != "" {
		multiOrder.Comments = comments
	}
	for _, item := range orderItems {
		seriesPart := item.Vm.VmProductID[0:4] // 'VM-A' part
		typePart := item.Vm.VmProductID[4:]    // 'x' part
		vmType, err := strconv.Atoi(typePart)
		if err != nil {
			// ... handle error
			panic(err)
		}
		vmType-- // packages are zero-indexed
		newVmOrder := NewVmOrder{
			Amount: NewVmOrderOption{
				VmCore:      0,
				VmDiskspace: 0,
				VmMem:       0,
				VmType:      vmType,
			},
			CustomDescription: "",
			// parent: "",
			Product: seriesPart,
			Provisioning: NewVmProvisioning{
				Hostname: item.Vm.Hostname,
				Os:       item.Vm.Os,
				// VmLocations: "f5bce822-ec48-68eb-68d9-38da8240580a",
			},
			Reference: RandomString(8),
		}
		multiOrder.Orders = append(multiOrder.Orders, newVmOrder)
	}

	rb, err := json.Marshal(multiOrder)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/account/v1/customer/self/order", c.HostURL), strings.NewReader(string(rb)))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	order := Order{}
	err = json.Unmarshal(body, &order)
	if err != nil {
		return nil, err
	}

	return &order, nil
}

// UpdateOrder - Updates an order
func (c *Client) UpdateOrder(orderID string, orderItems []OrderItem) (*Order, error) {
	rb, err := json.Marshal(orderItems)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/orders/%s", c.HostURL, orderID), strings.NewReader(string(rb)))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	order := Order{}
	err = json.Unmarshal(body, &order)
	if err != nil {
		return nil, err
	}

	return &order, nil
}

// DeleteOrder - Deletes an order
func (c *Client) DeleteOrder(orderID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/orders/%s", c.HostURL, orderID), nil)
	if err != nil {
		return err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return err
	}

	if string(body) != "Deleted order" {
		return errors.New(string(body))
	}

	return nil
}

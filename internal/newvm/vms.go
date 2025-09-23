package newvm

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type NewVmVmWrapper struct {
	Vm Vm `json:"vm"`
}

// GetVms - Returns list of available VM products (no auth required)
func (c *Client) GetVmProducts() ([]VmProduct, error) {
	vmProducts := []VmProduct{}

	type IntermediateProduct struct {
		ID               string                              `json:"id"`
		BasePrice        float64                             `json:"base_price"`
		Pricing          []IntermediatePricing               `json:"pricing"`
		Ram              int64                               `json:"ram"`
		Cores            int32                               `json:"cores"`
		HdSize           int64                               `json:"hdsize"`
		Price            float64                             `json:"price"`
		Properties       []IntermediateProperty              `json:"properties"`
		OptionProperties []IntermediateProductOptionProperty `json:"product_option_properties"`
	}

	intermediates := []IntermediateProduct{}

	// first we obtain all 'VM-A' products @hardcoded product ID
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/account/v1/product/VM-A", c.HostURL), nil)
	if err != nil {
		return nil, err
	} else {
		body, err := c.doRequest(req)
		if err != nil {
			return nil, err
		} else {
			intermediateProductVmA := IntermediateProduct{}
			err = json.Unmarshal(body, &intermediateProductVmA)
			if err != nil {
				return nil, err
			} else {
				intermediates = append(intermediates, intermediateProductVmA)
			}
		}
	}

	// and then we add all Vm-B products @hardcoded product ID
	req, err = http.NewRequest("GET", fmt.Sprintf("%s/account/v1/product/VM-B", c.HostURL), nil)
	if err != nil {
		return nil, err
	} else {
		body, err := c.doRequest(req)
		if err != nil {
			log.Printf("Error during request: %v\n", err)
			//return nil, err
		} else {
			intermediateProductVmB := IntermediateProduct{}
			err = json.Unmarshal(body, &intermediateProductVmB)
			if err != nil {
				return nil, err
			} else {
				intermediates = append(intermediates, intermediateProductVmB)
			}
		}
	}

	// loop the intermediate data and populate our final list of VM products
	for _, intermediate := range intermediates {
		var ramPropertyID string
		var coresPropertyID string
		var hdSizePropertyID string
		for _, property := range intermediate.Properties {
			switch property.Key {
			case "memory":
				ramPropertyID = property.ID
			case "cpu":
				coresPropertyID = property.ID
			case "diskspace":
				hdSizePropertyID = property.ID
			}
		}

		for _, pricing := range intermediate.Pricing {
			if pricing.ID == "vm_type" {
				for _, enumOption := range pricing.EnumOptions {
					var ram int64 = 0
					var cores int32 = 0
					var hdSize int64 = 0
					for _, optionProperty := range intermediate.OptionProperties {
						if optionProperty.PropertyID == ramPropertyID && optionProperty.PricingID == "vm_type" && optionProperty.Index == enumOption.Index {
							ram, err = strconv.ParseInt(optionProperty.Value, 10, 64)
							if err != nil {
								fmt.Println("Conversion error:", err)
								return []VmProduct{}, err
							}
						} else if optionProperty.PropertyID == coresPropertyID && optionProperty.PricingID == "vm_type" && optionProperty.Index == enumOption.Index {
							c, err := strconv.ParseInt(optionProperty.Value, 10, 32)
							if err != nil {
								fmt.Println("Conversion error:", err)
								return []VmProduct{}, err
							}
							cores = int32(c)
						} else if optionProperty.PropertyID == hdSizePropertyID && optionProperty.PricingID == "vm_type" && optionProperty.Index == enumOption.Index {
							hdSize, err = strconv.ParseInt(optionProperty.Value, 10, 64)
							if err != nil {
								fmt.Println("Conversion error:", err)
								return []VmProduct{}, err
							}
						}
					}

					vmProducts = append(vmProducts, VmProduct{
						ID:        enumOption.Name,
						ProductID: intermediate.ID,
						Ram:       ram,
						Cores:     cores,
						HdSize:    hdSize,
						Price:     intermediate.BasePrice + enumOption.Price,
					})
				}
			}
		}
	}

	return vmProducts, nil
}

// GetVm - Returns specific vm details
func (c *Client) GetVm(orderID string) (*Vm, error) {
	if orderID != "" {
		// combine data from various API paths
		// first obtain the order details
		reqOrder, err := http.NewRequest("GET", fmt.Sprintf("%s/account/v1/order/%s", c.HostURL, orderID), nil)
		if err != nil {
			return nil, err
		}
		bodyOrder, err := c.doRequest(reqOrder)
		if err != nil {
			return nil, err
		}
		orderData := NewVmOrderWrapper{}
		err = json.Unmarshal(bodyOrder, &orderData)
		if err != nil {
			return nil, err
		}

		vmType := ""
		vmRam := 0
		vmCores := 0
		vmHdSize := 0
		for _, orderOption := range orderData.Order.Options {
			switch orderOption.OptionID {
			case "vm_type":
				vmType = strconv.Itoa(orderOption.ItemCount + 1)
			case "vm_mem":
				vmRam = orderOption.ItemCount
			case "vm_core":
				vmCores = orderOption.ItemCount
			case "vm_diskspace":
				vmHdSize = orderOption.ItemCount
			}
		}

		// merge outstanding change requests with the order details if any
		if orderData.Order.NeedsChange == 1 {
			reqChanges, err := http.NewRequest("GET", fmt.Sprintf("%s/account/v1/order/changerequest?orderId=%s", c.HostURL, orderID), nil)
			if err != nil {
				return nil, err
			}
			bodyChanges, err := c.doRequest(reqChanges)
			if err != nil {
				return nil, err
			}
			changesData := NewVmChangeRequestsWrapper{}
			err = json.Unmarshal(bodyChanges, &changesData)
			if err != nil {
				return nil, err
			}
			changeRequest := changesData.Changes[0]
			newOptions := NewVmPricing{}
			if err := json.Unmarshal([]byte(changeRequest.NewOptions), &newOptions); err != nil {
				panic(err)
			}

			val := reflect.ValueOf(newOptions)
			typ := val.Type()
			for i := 0; i < val.NumField(); i++ {
				field := typ.Field(i)
				value := val.Field(i)

				// safety checks
				if !value.IsValid() || !value.CanInterface() {
					continue
				}

				// Convert any integer kind to int in a safe way
				var castVal int
				switch value.Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					castVal = int(value.Int())
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
					castVal = int(value.Uint())
				default:
					log.Printf("unsupported kind %s for field %s", value.Kind(), field.Name)
					continue
				}

				// Extract the JSON tag key without ,omitempty
				tagName := field.Tag.Get("json")
				if idx := strings.Index(tagName, ","); idx != -1 {
					tagName = tagName[:idx]
				}

				switch tagName {
				case "vm_type":
					vmType = strconv.Itoa(castVal + 1)
				case "vm_mem":
					vmRam = castVal
				case "vm_core":
					vmCores = castVal
				case "vm_diskspace":
					vmHdSize = castVal
				}
			}
		}

		operatingSystem := ""
		if orderData.Order.ProvisioningOptions.Provisioning.Os != "" {
			// obtain operating systems
			operatingSystems, err := c.GetOperatingSystems()
			if err != nil {
				return nil, err
			}
			for _, os := range operatingSystems {
				if os.ID == orderData.Order.ProvisioningOptions.Provisioning.Os {
					operatingSystem = os.Tag
					break
				}
			}
			log.Printf("Obtained operating system tag '%s' from ID <%s>", operatingSystem, orderData.Order.ProvisioningOptions.Provisioning.Os)
		}

		locationCode := ""
		if orderData.Order.ProvisioningOptions.Provisioning.Location != "" {
			// obtain locations
			locations, err := c.GetLocations()
			if err != nil {
				return nil, err
			}
			for _, location := range locations {
				if location.ID == orderData.Order.ProvisioningOptions.Provisioning.Location {
					locationCode = location.Code
					break
				}
			}
			log.Printf("Obtained location code '%s' from ID <%s>", locationCode, orderData.Order.ProvisioningOptions.Provisioning.Location)
		}

		var vpcNumber int32 = 0
		// obtain all VPC members and see if our order is in there
		// @todo support for multiple VPCs
		vpcMembers, err := c.GetVpcMembers()
		if err != nil {
			return nil, err
		}
		for _, vpcMember := range vpcMembers {
			if orderData.Order.ID == vpcMember.OrderId { // for now, we just use the first match
				vpcNumber = vpcMember.Vxlan
				break
			}
		}
		log.Printf("Obtained VPC number '%v' for order ID %v", vpcNumber, orderData.Order.ID)

		// populate the VM with obtained data values
		vm := Vm{
			ID:          orderData.Order.ProvisioningData.VmUuid,
			OrderID:     orderData.Order.ID,
			VmProductID: orderData.Order.ProductID + vmType, // eg. 'VM-A' + '2' becomes 'VM-A2'
			Hostname:    orderData.Order.ProvisioningOptions.Provisioning.Hostname,
			Os:          operatingSystem,
			Location:    locationCode,
			Ram:         int64(vmRam),    //orderData.Order.ProvisioningOptions.Pricing.Ram,
			Cores:       vmCores,         //int(orderData.Order.ProvisioningOptions.Pricing.Cores),
			HdSize:      int64(vmHdSize), //orderData.Order.ProvisioningOptions.Pricing.HdSize,
			SshKey:      orderData.Order.ProvisioningOptions.Provisioning.SshKey,
			IsVpcOnly:   orderData.Order.ProvisioningOptions.Provisioning.IsVpcOnly,
			UseDhcp:     orderData.Order.ProvisioningOptions.Provisioning.UseDhcp,
		}
		err = json.Unmarshal(bodyOrder, &vm)
		if err != nil {
			return nil, err
		}
		// log.Printf("VM: %+v\n", vm)

		return &vm, nil
	} else {
		vm := Vm{}
		return &vm, nil
	}
}

func getOperatingSystemID(c *Client, osTag string) (string, error) {
	log.Printf("Looking up operating system ID for tag '%s'", osTag)
	operatingSystemID := ""
	if osTag != "" {
		// obtain operating systems
		operatingSystems, err := c.GetOperatingSystems()
		if err != nil {
			return "", err
		}
		for _, os := range operatingSystems {
			if os.Tag == osTag {
				operatingSystemID = os.ID
				break
			}
		}
		log.Printf("Obtained operating system ID <%s> for tag '%s'", operatingSystemID, osTag)
	}

	return operatingSystemID, nil
}

func getLocationID(c *Client, locationCode string) (string, error) {
	log.Printf("Looking up location ID for code '%s'", locationCode)
	locationID := ""
	if locationCode != "" {
		locations, err := c.GetLocations()
		if err != nil {
			return "", err
		}
		for _, location := range locations {
			if location.Code == locationCode {
				locationID = location.ID
				break
			}
		}
		log.Printf("Obtained location ID <%s> for code '%s'", locationID, locationCode)
	}

	return locationID, nil
}

func getVxlanID(c *Client, vpcNumber int32) (string, error) {
	log.Printf("Looking up VxLAN ID for number '%d'", vpcNumber)
	vxlanID := ""
	if vpcNumber > 0 {
		vxlans, err := c.GetVpcs()
		if err != nil {
			return "", err
		}
		for _, vxlan := range vxlans {
			if vxlan.Number == vpcNumber {
				vxlanID = vxlan.ID
				break
			}
		}
		log.Printf("Obtained VxLAN ID <%s> for number '%d'", vxlanID, vpcNumber)
	}

	return vxlanID, nil
}

func splitVmProductID(vmProductID string) (string, int, error) {
	productCode := vmProductID[0:4] // 'VM-A' part
	typePart := vmProductID[4:]     // 'x' part
	vmType, err := strconv.Atoi(typePart)
	if err != nil {
		return "", 0, err // ... handle error
	}
	vmType-- // packages are zero-indexed

	return productCode, vmType, nil
}

// CreateVm - Create new vm order
func (c *Client) CreateVm(vm Vm) (*Vm, error) {
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
		VxlanID     string `json:"vxlanid,omitempty"`
		SshKey      string `json:"sshkey,omitempty"`
		IsVpcOnly   bool   `json:"isVpcOnly,omitempty"`
		UseDhcp     bool   `json:"useDhcp,omitempty"`
		IpAddress   string `json:"ipaddress,omitempty"`
		SubnetMask  string `json:"subnetmask,omitempty"`
		Gateway     string `json:"gateway,omitempty"`
		DnsServer   string `json:"dnsserver,omitempty"`
	}
	type NewVmOrder struct {
		Amount            NewVmOrderOption  `json:"amount,omitempty"`
		CustomDescription string            `json:"custom_description,omitempty"`
		Parent            string            `json:"parentid,omitempty"`
		Provisioning      NewVmProvisioning `json:"provisioning,omitempty"`
		AutoProvision     bool              `json:"autoProvision,omitempty"`
		FinishOrderGroup  bool              `json:"finishOrderGroup,omitempty"`
		PromoCodes        []string          `json:"promoCodes,omitempty"`
	}
	// split vm product ID to get product code and type
	productCode, vmType, err := splitVmProductID(vm.VmProductID)
	if err != nil {
		panic(err) // ... handle error
	}
	// get operating system ID
	osID, err := getOperatingSystemID(c, vm.Os)
	if err != nil {
		return nil, err
	}
	// get location ID
	locationID, err := getLocationID(c, vm.Location)
	if err != nil {
		return nil, err
	}
	// get VxLAN ID
	vxlanID, err := getVxlanID(c, vm.Vpc)
	if err != nil {
		return nil, err
	}

	provisioning := NewVmProvisioning{
		Hostname: vm.Hostname,
		Os:       osID,
	}
	if locationID != "" {
		provisioning.VmLocations = locationID
	}
	if vxlanID != "" {
		provisioning.VxlanID = vxlanID
	}
	if vm.SshKey != "" {
		provisioning.SshKey = vm.SshKey
	}
	if vm.IsVpcOnly {
		provisioning.IsVpcOnly = true
	}
	if vm.UseDhcp {
		provisioning.UseDhcp = true
	} else {
		provisioning.IpAddress = vm.IpAddress
		provisioning.SubnetMask = vm.SubnetMask
		provisioning.Gateway = vm.Gateway
		provisioning.DnsServer = vm.DnsServer
	}

	newVmOrder := NewVmOrder{
		Amount: NewVmOrderOption{
			VmCore:      vm.Cores,
			VmDiskspace: int(vm.HdSize),
			VmMem:       int(vm.Ram),
			VmType:      vmType,
		},
		CustomDescription: "",
		// parent: "",
		Provisioning:     provisioning,
		AutoProvision:    true,
		FinishOrderGroup: true,
	}

	rb, err := json.Marshal(newVmOrder)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/account/v1/customer/self/order/%s", c.HostURL, productCode), strings.NewReader(string(rb)))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	type Result struct {
		OrderID int `json:"orderid"`
	}
	var responseBody Result
	err = json.Unmarshal(body, &responseBody)
	if err != nil {
		return nil, err
	}

	vm.OrderID = responseBody.OrderID
	return &vm, nil
}

// UpdateVm - Updates an order
func (c *Client) UpdateVm(orderID string, vm Vm) (*Vm, error) {
	// Order @NewVM Change request structure
	type NewVmChangeOption struct {
		VmCore      int `json:"vm_core"`
		VmDiskspace int `json:"vm_diskspace"`
		VmMem       int `json:"vm_mem"`
		VmType      int `json:"vm_type"`
	}
	// @todo support custom.hardDiskSizes
	type NewVmChangeRequest struct {
		Options NewVmChangeOption `json:"options"`
	}

	// split vm product ID to get product code and type
	_, vmType, err := splitVmProductID(vm.VmProductID) // can also be: productCode, vmType, err :=
	if err != nil {
		panic(err) // ... handle error
	}

	newVmChange := NewVmChangeRequest{
		Options: NewVmChangeOption{
			VmCore:      vm.Cores,
			VmDiskspace: int(vm.HdSize),
			VmMem:       int(vm.Ram),
			VmType:      vmType,
		},
	}
	rb, err := json.Marshal(newVmChange)
	if err != nil {
		return nil, err
	}
	// change request
	reqChange, err := http.NewRequest("PUT", fmt.Sprintf("%s/account/v1/order/%s", c.HostURL, orderID), strings.NewReader(string(rb)))
	if err != nil {
		return nil, err
	}

	resChange, err := c.doRequest(reqChange)
	if err != nil {
		return nil, err
	}

	vmOrder := Vm{}
	err = json.Unmarshal(resChange, &vmOrder)
	if err != nil {
		return nil, err
	}

	return &vmOrder, nil
}

// DeleteVm - Deletes a VM
func (c *Client) DeleteVm(orderID string) error {
	// obtain VM uuid
	reqOrder, err := http.NewRequest("GET", fmt.Sprintf("%s/account/v1/order/%s", c.HostURL, orderID), nil)
	if err != nil {
		return err
	}
	bodyOrder, err := c.doRequest(reqOrder)
	if err != nil {
		return err
	}
	var orderData NewVmOrderWrapper
	err = json.Unmarshal(bodyOrder, &orderData)
	if err != nil {
		return err
	}
	log.Printf("Obtained VM uuid: %s", orderData.Order.ProvisioningData.VmUuid)
	log.Printf("Obtained Billed until: %s", orderData.Order.BilledUntil)

	if orderData.Order.ProvisioningData.VmUuid != "" {
		// get current state of VM
		reqState, err := http.NewRequest("GET", fmt.Sprintf("%s/backend/com.newvm.network/v1/vm/%s", c.HostURL, orderData.Order.ProvisioningData.VmUuid), nil)
		if err != nil {
			return err
		}
		bodyState, err := c.doRequest(reqState)
		if err != nil {
			return err
		}
		var stateData NewVmVmWrapper
		err = json.Unmarshal(bodyState, &stateData)
		if err != nil {
			return err
		}

		if stateData.Vm.Status == "STOPPED" {
			log.Printf("VM %s state is already 'STOPPED'", orderID)
		} else {
			// turn off VM if not off already
			reqTurnOff, err := http.NewRequest("PATCH", fmt.Sprintf("%s/backend/com.newvm.network/v1/vm2/%s/changeState/off", c.HostURL, orderData.Order.ProvisioningData.VmUuid), nil)
			if err != nil {
				return err
			}
			_, err = c.doRequest(reqTurnOff)
			if err != nil {
				return err
			}
			log.Printf("Turned off VM %s", orderID)
		}
	}

	// set end date for order
	type NewVmOrderEnd struct {
		EndDate          string `json:"end_date"`
		IncludeSubOrders bool   `json:"includeSubOrders,omitempty"`
	}
	endDate, err := time.Parse("2006-01-02T15:04:05.000Z07:00", orderData.Order.BilledUntil) // this is the format for RFC3339 including milliseconds
	if err != nil {
		panic(err)
	}
	timezone, err := time.LoadLocation("Europe/Amsterdam")
	if err != nil {
		panic(err)
	}
	newVmOrderEnd := NewVmOrderEnd{
		EndDate:          endDate.In(timezone).Format("2006-01-02"), // this is the format for YYYY-MM-DD
		IncludeSubOrders: true,
	}
	reqBodyOrderEnd, err := json.Marshal(newVmOrderEnd)
	if err != nil {
		return err
	}
	reqOrderEnd, err := http.NewRequest("PUT", fmt.Sprintf("%s/account/v1/order/%s/enddate", c.HostURL, orderID), strings.NewReader(string(reqBodyOrderEnd)))
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
	log.Printf("Set end date for order %s", orderID)

	// @todo also delete sub orders
	// @todo also delete sub orders
	// @todo also delete sub orders

	return nil
}

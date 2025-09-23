package newvm

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type NewVmControlPanelWrapper struct {
	ControlPanel ControlPanel `json:"control_panel"`
}

// GetControlPanelProducts - Returns list of available control panel products (no auth required)
func (c *Client) GetControlPanelProducts() ([]ControlPanelProduct, error) {
	controlPanelProducts := []ControlPanelProduct{}

	type IntermediateProduct struct {
		ID          string                `json:"id"`
		Description string                `json:"description"`
		BasePrice   float64               `json:"base_price"`
		Pricing     []IntermediatePricing `json:"pricing"`
		Price       float64               `json:"default_price"`
	}

	intermediates := []IntermediateProduct{}

	// first we obtain all DirectAdmin products @hardcoded product ID
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/account/v1/product/CP_DIRECTADMIN", c.HostURL), nil)
	if err != nil {
		return nil, err
	} else {
		body, err := c.doRequest(req)
		if err != nil {
			return nil, err
		} else {
			intermediateProductDirectAdmin := IntermediateProduct{}
			err = json.Unmarshal(body, &intermediateProductDirectAdmin)
			if err != nil {
				return nil, err
			} else {
				intermediates = append(intermediates, intermediateProductDirectAdmin)
			}
		}
	}

	// and then we add all Plesk products @hardcoded product ID
	req, err = http.NewRequest("GET", fmt.Sprintf("%s/account/v1/product/CP_PLESK", c.HostURL), nil)
	if err != nil {
		return nil, err
	} else {
		body, err := c.doRequest(req)
		if err != nil {
			log.Printf("Error during request: %v\n", err)
			//return nil, err
		} else {
			intermediateProductPlesk := IntermediateProduct{}
			err = json.Unmarshal(body, &intermediateProductPlesk)
			if err != nil {
				return nil, err
			} else {
				intermediates = append(intermediates, intermediateProductPlesk)
			}
		}
	}

	// loop the intermediate product data and populate our final list of control panel products
	for _, intermediate := range intermediates {
		extensions := []ControlPanelExtension{}
		// first loop is to obtain all extensions
		for _, pricing := range intermediate.Pricing {
			if pricing.Type == "flag" {
				extensions = append(extensions, ControlPanelExtension{
					ID:          pricing.ID,
					Description: pricing.Description,
					Price:       pricing.DefaultPrice,
				})
			}
		}
		// then loop to create the products
		for _, pricing := range intermediate.Pricing {
			if pricing.ID == "da_license" || pricing.ID == "plesk_12_license" { // @hardcoded
				for _, enumOption := range pricing.EnumOptions {
					controlPanelProducts = append(controlPanelProducts, ControlPanelProduct{
						ID:          intermediate.ID + "." + pricing.ID + "." + strconv.FormatInt(int64(enumOption.Index), 10),
						Type:        enumOption.Name,
						Description: intermediate.Description + ": " + enumOption.Description,
						Price:       intermediate.BasePrice + enumOption.Price,
						Extensions:  extensions,
					})
				}
			}
		}
	}

	return controlPanelProducts, nil
}

// GetControlPanel - Returns specific control panel details
func (c *Client) GetControlPanel(orderID int64) (*ControlPanel, error) {
	if orderID > 0 {
		// combine data from various API paths
		// first obtain the order details
		reqOrder, err := http.NewRequest("GET", fmt.Sprintf("%s/account/v1/order/%s", c.HostURL, strconv.FormatInt(orderID, 10)), nil)
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

		licenseType := ""
		extensions := []ControlPanelExtension{}
		for _, orderOption := range orderData.Order.Options {
			switch orderOption.OptionID {
			case "da_license":
			case "plesk_12_license":
				licenseType = orderOption.OptionID + "." + strconv.Itoa(orderOption.ItemCount)
			}

			if orderOption.Type == "flag" && orderOption.ItemCount == 1 {
				switch orderOption.OptionID {
				case "plesk_extension_dnssec":
				case "plesk_extension_hosting_pack":
				case "plesk_extension_powerpack":
				case "plesk_extension_language_pack":
					extensions = append(extensions, ControlPanelExtension{
						ID:          orderOption.OptionID,
						Description: orderOption.Description,
					})
				}

			}
		}

		// merge outstanding change requests with the order details if any
		if orderData.Order.NeedsChange == 1 {
			reqChanges, err := http.NewRequest("GET", fmt.Sprintf("%s/account/v1/order/changerequest?orderId=%s", c.HostURL, strconv.FormatInt(orderID, 10)), nil)
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
		}

		// populate the control panel with obtained data values
		controlPanel := ControlPanel{
			ID:         orderData.Order.ID,
			VmID:       orderData.Order.ParentID,
			ProductID:  orderData.Order.ProductID + "." + licenseType,
			Extensions: extensions,
		}
		err = json.Unmarshal(bodyOrder, &controlPanel)
		if err != nil {
			return nil, err
		}

		return &controlPanel, nil
	} else {
		controlPanel := ControlPanel{}
		return &controlPanel, nil
	}
}

// split the control panel product ID into 3 parts (example: CP_PLESK.plesk_12_license.1)
func splitControlPanelProductID(controlPanelProductID string) (string, int) {
	parts := strings.Split(controlPanelProductID, ".")
	productCode := parts[0] // 'CP_PLESK' part
	//enumOptionPart := parts[1]                   // 'plesk_12_license' part (unused)
	enumIndexPart, err := strconv.Atoi(parts[2]) // '1' part
	if err != nil {
		panic(err)
	}

	return productCode, enumIndexPart
}

// CreateControlPanel - Create new control panel order
func (c *Client) CreateControlPanel(controlPanel ControlPanel) (*ControlPanel, error) {
	// Order @NewVM Order structure
	type NewVmOrderOption struct {
		DirectAdminLicense int `json:"da_license,omitempty"`
		PleskLicense       int `json:"plesk_12_license,omitempty"`
		PleskDnssec        int `json:"plesk_extension_dnssec,omitempty"`
		PleskHostingPack   int `json:"plesk_extension_hosting_pack,omitempty"`
		PleskPowerPack     int `json:"plesk_extension_powerpack,omitempty"`
		PleskLanguagePack  int `json:"plesk_extension_language_pack,omitempty"`
	}
	type NewVmProvisioning struct {
		DirectAdminLicense  string `json:"da_license,omitempty"`
		DirectAdminUrl      string `json:"da_url,omitempty"`
		DirectAdminUser     string `json:"da_user,omitempty"`
		DirectAdminPassword string `json:"da_admin,omitempty"`
		PleskLicense        string `json:"plesk_license,omitempty"`
		PleskCode           string `json:"plesk_code,omitempty"`
		PleskKnownIP        string `json:"known_ip,omitempty"`
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

	// split control panel product ID to get product code and license type
	productCode, licenseType := splitControlPanelProductID(controlPanel.ProductID)
	orderOption := NewVmOrderOption{}
	if productCode == "CP_DIRECTADMIN" { // @hardcoded
		orderOption.DirectAdminLicense = licenseType
	} else { // CP_PLESK
		orderOption.PleskLicense = licenseType
	}
	for _, extension := range controlPanel.Extensions {
		switch extension.ID {
		case "plesk_extension_dnssec":
			orderOption.PleskDnssec = 1
		case "plesk_extension_hosting_pack":
			orderOption.PleskHostingPack = 1
		case "plesk_extension_powerpack":
			orderOption.PleskPowerPack = 1
		case "plesk_extension_language_pack":
			orderOption.PleskLanguagePack = 1
		}
	}
	newVmOrder := NewVmOrder{
		Amount:            orderOption,
		CustomDescription: "",
		Parent:            strconv.FormatInt(int64(controlPanel.VmID), 10),
		//Provisioning:     provisioning,
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

	controlPanel.ID = int(responseBody.OrderID)
	return &controlPanel, nil
}

// UpdateControlPanel - Updates an order
func (c *Client) UpdateControlPanel(orderID int64, controlPanel ControlPanel) (*ControlPanel, error) {
	// Order @NewVM Change request structure
	type NewVmChangeOption struct {
		DirectAdminLicense int `json:"da_license,omitempty"`
		PleskLicense       int `json:"plesk_12_license,omitempty"`
		PleskDnssec        int `json:"plesk_extension_dnssec,omitempty"`
		PleskHostingPack   int `json:"plesk_extension_hosting_pack,omitempty"`
		PleskLanguagePack  int `json:"plesk_extension_language_pack,omitempty"`
		PleskPowerPack     int `json:"plesk_extension_powerpack,omitempty"`
	}
	// @todo support custom.hardDiskSizes
	type NewVmChangeRequest struct {
		Options NewVmChangeOption `json:"options"`
	}

	newVmChange := NewVmChangeRequest{
		Options: NewVmChangeOption{},
	}
	productCode, licenseType := splitControlPanelProductID(controlPanel.ProductID)
	if productCode == "CP_DIRECTADMIN" { // @hardcoded
		newVmChange.Options.DirectAdminLicense = licenseType
	} else { // CP_PLESK
		newVmChange.Options.PleskLicense = licenseType
	}
	for _, extension := range controlPanel.Extensions {
		switch extension.ID {
		case "plesk_extension_dnssec":
			newVmChange.Options.PleskDnssec = 1
		case "plesk_extension_hosting_pack":
			newVmChange.Options.PleskHostingPack = 1
		case "plesk_extension_powerpack":
			newVmChange.Options.PleskPowerPack = 1
		case "plesk_extension_language_pack":
			newVmChange.Options.PleskLanguagePack = 1
		}
	}

	rb, err := json.Marshal(newVmChange)
	if err != nil {
		return nil, err
	}
	// change request
	reqChange, err := http.NewRequest("PUT", fmt.Sprintf("%s/account/v1/order/%s", c.HostURL, strconv.FormatInt(orderID, 10)), strings.NewReader(string(rb)))
	if err != nil {
		return nil, err
	}

	resChange, err := c.doRequest(reqChange)
	if err != nil {
		return nil, err
	}

	controlPanelOrder := ControlPanel{}
	err = json.Unmarshal(resChange, &controlPanelOrder)
	if err != nil {
		return nil, err
	}

	return &controlPanelOrder, nil
}

// DeleteControlPanel - Deletes a control panel
func (c *Client) DeleteControlPanel(orderID int64) error {
	// obtain VM uuid
	reqOrder, err := http.NewRequest("GET", fmt.Sprintf("%s/account/v1/order/%s", c.HostURL, strconv.FormatInt(orderID, 10)), nil)
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
	log.Printf("Obtained Billed until: %s", orderData.Order.BilledUntil)

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
	reqOrderEnd, err := http.NewRequest("PUT", fmt.Sprintf("%s/account/v1/order/%s/enddate", c.HostURL, strconv.FormatInt(orderID, 10)), strings.NewReader(string(reqBodyOrderEnd)))
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
	log.Printf("Set end date for order %s", strconv.FormatInt(orderID, 10))

	// @todo also delete sub orders
	// @todo also delete sub orders
	// @todo also delete sub orders

	return nil
}

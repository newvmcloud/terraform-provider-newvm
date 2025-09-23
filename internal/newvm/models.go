package newvm

// Control Panel
type ControlPanel struct {
	ID         int                     `json:"order_id,omitempty"`
	VmID       int                     `json:"vm_id,omitempty"`
	ProductID  string                  `json:"product,omitempty"` /* eg. CP_PLESK.plesk_12_license.1 */
	Extensions []ControlPanelExtension `json:"extensions,omitempty"`
}

// Control Panel extension
type ControlPanelExtension struct {
	ID          string  `json:"id"`
	Description string  `json:"licenseType"`
	Price       float64 `json:"price"`
}

// Control Panel product
type ControlPanelProduct struct {
	ID          string                  `json:"id"` /* eg. CP_PLESK.plesk_12_license.1 */
	Type        string                  `json:"licenseType"`
	Price       float64                 `json:"price"`
	Description string                  `json:"description"`
	Extensions  []ControlPanelExtension `json:"extensions"`
}

type IntermediateEnumOption struct {
	Index       int     `json:"enum_index"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type IntermediatePricing struct {
	ID           string                       `json:"id"`
	Description  string                       `json:"description"`
	Type         string                       `json:"type"`
	Unit         string                       `json:"unit"`
	Minimum      int                          `json:"min"`
	Maximum      int                          `json:"max"`
	EnumOptions  []IntermediateEnumOption     `json:"enum_options"`
	Pricing      []IntermediatePricingPricing `json:"pricing"`
	DefaultPrice float64                      `json:"default_price"`
}

type IntermediatePricingPricing struct {
	Minimum   int     `json:"min"`
	Price     float64 `json:"price"`
	Increment int     `json:"per"`
	UnitPrice float64 `json:"unit_price"`
}

type IntermediateProductOptionProperty struct {
	Index      int    `json:"optionindex"`
	PropertyID string `json:"property_id"`
	PricingID  string `json:"product_option_id"`
	Value      string `json:"value"`
}

type IntermediateProperty struct {
	ID    string `json:"id"`
	Key   string `json:"key"`
	Label string `json:"label"`
	Unit  string `json:"unit"`
}

// NewVM location
type Location struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Code          string   `json:"extcode"`
	ProductIds    []string `json:"productIds,omitempty"`
	Provisionable int      `json:"provisionable,omitempty"`
}

// NewVM change request
type NewVmChangeRequest struct {
	ID                int    `json:"id"`
	OrderID           int    `json:"order_id"`
	ScheduledDate     string `json:"scheduled_date,omitempty"`
	NewOptions        string `json:"new_option"`
	ProvisioningError string `json:"provisionerror,omitempty"`
	IsInternalError   int    `json:"isprovisionerrorinternal"`
}

type NewVmChangeRequestsWrapper struct {
	Changes []NewVmChangeRequest `json:"result"`
}

// NewVM provisioning option new style
type NewVmOption struct {
	OrderID     int      `json:"orderid,omitempty"`
	OptionID    string   `json:"option_id,omitempty"`
	ItemCount   int      `json:"item_count,omitempty"`
	Description string   `json:"description,omitempty"`
	Unit        string   `json:"unit,omitempty"`
	Type        string   `json:"type,omitempty"`
	IsSetup     int      `json:"is_setup,omitempty"`
	Meta        []string `json:"meta,omitempty"`
}

// NewVM order
type NewVmOrder struct {
	ID                  int                      `json:"id"`
	ParentID            int                      `json:"parentid"`
	ProductID           string                   `json:"product_id"`
	Options             []NewVmOption            `json:"options"`
	ProvisioningOptions NewVmProvisioningOptions `json:"prov_options"`
	ProvisioningData    NewVmProvisioningData    `json:"prov_data"`
	BilledUntil         string                   `json:"billed_until,omitempty"`
	NeedsChange         int                      `json:"needs_change,omitempty"`
}

type NewVmOrderWrapper struct {
	Order NewVmOrder `json:"order"`
}

// NewVM pricing option
type NewVmPricing struct {
	Type   int32 `json:"vm_type,omitempty"`
	Ram    int64 `json:"vm_mem,omitempty"`
	Cores  int32 `json:"vm_core,omitempty"`
	HdSize int64 `json:"vm_diskspace,omitempty"`
}

// NewVM provisioning option
type NewVmProvisioning struct {
	Hostname   string `json:"hostname,omitempty"`
	SshKey     string `json:"sshkey,omitempty"`
	VxlanId    string `json:"vxlanid,omitempty"`
	Os         string `json:"os,omitempty"`
	Location   string `json:"vm_locations,omitempty"`
	IsVpcOnly  bool   `json:"isVpcOnly,omitempty"`
	UseDhcp    bool   `json:"useDhcp,omitempty"`
	IpAddress  string `json:"ipaddress,omitempty"`
	SubnetMask string `json:"subnetmask,omitempty"`
	Gateway    string `json:"gateway,omitempty"`
	DnsServer  string `json:"dnsserver,omitempty"`
}

// NewVM provisioning data
type NewVmProvisioningData struct {
	VmUuid      string `json:"vm_uuid,omitempty"`
	VmIpAddress string `json:"vm_ipaddress,omitempty"`
	VmRootUser  string `json:"vm_rootuser,omitempty"`
	VmPassword  string `json:"vm_password,omitempty"`
}

// NewVM provisioning options
type NewVmProvisioningOptions struct {
	Pricing       NewVmPricing      `json:"amount,omitempty"`
	Provisioning  NewVmProvisioning `json:"provisioning,omitempty"`
	AutoProvision bool              `json:"auto_provision,omitempty"`
	Comments      string            `json:"comments,omitempty"`
}

// NewVM OS
type OperatingSystem struct {
	ID   string `json:"id"`
	Tag  string `json:"idtag"`
	Name string `json:"name"`
	// Purpose                string `json:"purpose,omitempty"`
	Platform string `json:"platform,omitempty"`
	// Distro                 string `json:"distro,omitempty"`
	// Version                string `json:"version,omitempty"`
	// Firmware               string `json:"firmware,omitempty"`
	// InstallerIdentifier    string `json:"installeridentifier,omitempty"`
	// InstallImageIdentifier string `json:"installimageidentifier,omitempty"`
	// IsLegacy               int    `json:"legacy,omitempty"`
	// HasSshKeySupport       int    `json:"sshkeysupport,omitempty"`
	// HasSecureBootSupport   int    `json:"hassecurebootsupport,omitempty"`
	// HasFqdnHostnameSupport int    `json:"hasfqdnhostnamesupport,omitempty"`
	// MaxHostnameLength      int    `json:"maxhostnamelength,omitempty"`
	// AdminUsername          string `json:"adminusername,omitempty"`
}

// Order
type Order struct {
	ID       int         `json:"id,omitempty"`
	Comments string      `json:"comments,omitempty"`
	Items    []OrderItem `json:"items,omitempty"`
}

// OrderItem
type OrderItem struct {
	Vm       Vm  `json:"vm"`
	Quantity int `json:"quantity"`
}

// Product
type Product struct {
	ID                string  `json:"id"`
	Class             string  `json:"class,omitempty"`
	Description       string  `json:"description,omitempty"`
	Recurring         bool    `json:"recur,omitempty"`
	NeedsProvisioning bool    `json:"needs_prov,omitempty"`
	BasePrice         float64 `json:"base_price,omitempty"`
}

// VM
type Vm struct {
	ID                   string `json:"id"`
	OrderID              int    `json:"order_id,omitempty"`
	VmProductID          string `json:"product,omitempty"` /* eg. VM-A2 or VM-B5 */
	Os                   string `json:"os,omitempty"`
	Hostname             string `json:"hostname,omitempty"`
	Status               string `json:"status"`
	VmName               string `json:"vmname"`
	Location             string `json:"location,omitempty"`
	Ram                  int64  `json:"ram"`
	Reserved             int64  `json:"reserved"`
	Cores                int    `json:"cores"`
	HdSize               int64  `json:"hdsize"`
	BiosGuid             string `json:"biosGuid,omitempty"`
	IsSecureBootEnabled  bool   `json:"isSecureBootEnabled"`
	SecureBootTemplateId string `json:"secureBootTemplateId,omitempty"`
	Firmware             string `json:"firmware"`
	SshKey               string `json:"sshkey,omitempty"`
	IsVpcOnly            bool   `json:"isVpcOnly,omitempty"`
	UseDhcp              bool   `json:"useDhcp,omitempty"`
	Vpc                  int32  `json:"vpc,omitempty"`
	IpAddress            string `json:"ipaddress,omitempty"`
	SubnetMask           string `json:"subnetmask,omitempty"`
	Gateway              string `json:"gateway,omitempty"`
	DnsServer            string `json:"dnsserver,omitempty"`
}

// VM product
type VmProduct struct {
	ID        string  `json:"id"`                /* eg. VM-A5 or VM-B3 */
	ProductID string  `json:"product,omitempty"` /* VM-A or VM-B */
	Ram       int64   `json:"ram"`
	Cores     int32   `json:"cores"`
	HdSize    int64   `json:"hdsize"`
	Price     float64 `json:"price"`
}

// NewVM VPC
type Vpc struct {
	ID        string `json:"id"`
	Number    int32  `json:"vxlan"`
	Name      string `json:"label"`
	OwnerId   int32  `json:"ownerid,omitempty"`
	Removable int    `json:"removablebycustomer,omitempty"`
}

// NewVM VPC Member
type VpcMember struct {
	ID         string `json:"id"`
	Vxlan      int32  `json:"vxlan"`
	MacAddress string `json:"macaddress"`
	OrderId    int    `json:"orderid,omitempty"`
}

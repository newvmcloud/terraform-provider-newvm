terraform {
	required_providers {
		newvm = {
			source  = "newvmcloud/newvm"
			version = ">= 1.0.0"
		}
	}
}

provider "newvm" {
    username = var.username
    password = var.password
    host     = var.host
}

data "newvm_locations" "all" {}
data "newvm_operating_systems" "all" {}
data "newvm_vm_products" "all" {}

output "locations" {
    value = data.newvm_locations.all.list
}
output "operating_systems" {
    value = data.newvm_operating_systems.all.list
}
output "vm_products" {
    value = data.newvm_vm_products.all.list
}

// define resources
resource "newvm_vm" "tf_example" {
    product  = "VM-A2"
    os       = "win2022std"
    hostname = "tf-example.domain.tld"
    cores    = 2
    ram      = 2
    disk     = 20
}

terraform {
  required_version = ">= 1.5"

  required_providers {
    grouper = {
      source  = "notlimahrelyt/grouper"
      version = "~> 0.1"
    }
  }
}

# base_url, username, and password come from GROUPER_BASE_URL,
# GROUPER_USERNAME, and GROUPER_PASSWORD.
provider "grouper" {}

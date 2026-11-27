terraform {
  required_providers {
    grouper = {
      source = "notlimahrelyt/grouper"
    }
  }
}

# Credentials are read from GROUPER_USERNAME and GROUPER_PASSWORD in the
# environment; base_url may be set here or via GROUPER_BASE_URL.
provider "grouper" {
  base_url = "https://grouper.example.edu/grouper-ws/servicesRest"

  # request_timeout      = "30s"
  # insecure_skip_verify = false
}

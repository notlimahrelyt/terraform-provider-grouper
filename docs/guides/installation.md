---
page_title: "Installing the provider - Grouper Provider"
subcategory: ""
description: |-
  How to install the Grouper provider from the Terraform Registry or from a release download.
---

# Installing the provider

## From the Terraform Registry

Declare the provider and run `terraform init`; Terraform downloads it
automatically.

```hcl
terraform {
  required_providers {
    grouper = {
      source  = "notlimahrelyt/grouper"
      version = "~> 0.1"
    }
  }
}

provider "grouper" {
  base_url = "https://grouper.example.edu/grouper-ws/servicesRest"
  # GROUPER_USERNAME / GROUPER_PASSWORD come from the environment
}
```

```sh
export GROUPER_USERNAME="terraform-svc"
export GROUPER_PASSWORD="…"
terraform init
terraform plan
```

## From a release download (air-gapped or no registry access)

Each [GitHub release](https://github.com/notlimahrelyt/terraform-provider-grouper/releases)
carries pre-built zips, so no Go toolchain is needed.

### 1. Download the build for your machine

| Your machine      | File                                                 |
| ----------------- | ---------------------------------------------------- |
| Apple Silicon Mac | `terraform-provider-grouper_<VER>_darwin_arm64.zip`  |
| Intel Mac         | `terraform-provider-grouper_<VER>_darwin_amd64.zip`  |
| Linux (x86-64)    | `terraform-provider-grouper_<VER>_linux_amd64.zip`   |
| Linux (ARM)       | `terraform-provider-grouper_<VER>_linux_arm64.zip`   |
| Windows (x86-64)  | `terraform-provider-grouper_<VER>_windows_amd64.zip` |

`terraform version` prints your `os_arch` (e.g. `darwin_arm64`).

Verify the download against the release's `SHA256SUMS` file (and its `.sig`
signature) before installing.

### 2. Unpack it into the local plugin directory

```sh
VER=0.1.0                 # the release version, without the leading "v"
OS_ARCH=darwin_arm64      # from the table above / `terraform version`

DEST=~/.terraform.d/plugins/registry.terraform.io/notlimahrelyt/grouper/${VER}/${OS_ARCH}
mkdir -p "$DEST"
unzip ~/Downloads/terraform-provider-grouper_${VER}_${OS_ARCH}.zip -d "$DEST"

# macOS marks downloaded binaries as quarantined; clear it so Terraform can run it.
xattr -dr com.apple.quarantine "$DEST" 2>/dev/null || true
```

Terraform checks this directory before the registry, so the same
`required_providers` block above works unchanged. If `terraform init` cannot
find the provider, check that the version and `os_arch` folder names match
exactly.

### Sharing a mirror across a team

Put the unpacked layout on a shared location and point Terraform at it once per
machine via `~/.terraformrc`:

```hcl
provider_installation {
  filesystem_mirror {
    path    = "/path/to/shared/terraform-providers"
    include = ["notlimahrelyt/grouper"]
  }
  direct {
    exclude = ["notlimahrelyt/grouper"]
  }
}
```

The shared directory uses the same
`registry.terraform.io/notlimahrelyt/grouper/<version>/<os_arch>/` structure as
above. The release zips can also back a
[network mirror](https://developer.hashicorp.com/terraform/internals/provider-network-mirror-protocol)
or a private registry.

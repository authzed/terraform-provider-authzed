# Terraform Provider for Platform API

A Terraform provider for managing Platform API resources.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.23.4
- Access to Platform API server

### Clone the repository:
```git clone https://github.com/authzed/terraform-provider-platform-api.git```

### Build the provider:
```go build -o terraform-provider-platform-api```

### Install the provider locally:
mkdir -p ~/.terraform.d/plugins/local.providers/local/platform-api/1.0.0/darwin_arm64
cp terraform-provider-platform-api ~/.terraform.d/plugins/local.providers/local-platform-api/1.0.0/darwin_arm64/

## Project structure:
#### The layout of the repository is as follows:
```terraform-provider-platform-api/
├── examples/
│   └── provider/
│       └── provider.tf  # Example provider configuration
├── internal/
│   ├── client/          # Generated API client
│   │   ├── client.go
│   │   ├── configuration.go
│   │   ├── response.go
│   │   └── api_default.go
│   └── provider/        # Provider implementation
│       ├── provider.go
│       └── hello_resource.go
├── openapi-spec.yaml    # API specification
└── go.mod               # Go module dependencies
```

## Using the provider:
### Configure the required provider:
```hcl
  terraform {
  required_providers {
    platform-api = {
      source = "local.providers/local-platform-api"
      version = "1.0.0"
    }
  }
}
```

### Configure the platform-api provider:
```hcl
  provider "platform-api" {
  host  = "http://localhost:3030"
  token = "your-token"
```
### Define a resource:
```hcl
  resource "platform-api_hello" "example" {
  name = "World"
}
```

### Output the resource's response:
```hcl
  output "hello_response" {
  value = platform-api_hello.example.response
}
```

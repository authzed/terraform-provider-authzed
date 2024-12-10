terraform {
  required_providers {
    platform-api = {
      source = "local.providers/local/platform-api"
      version = "1.0.0"
    }
  }
}

provider "platform-api" {
  host = "http://localhost:3030"
  token = "testing123"
}

resource "platform-api_hello" "example" {
  name = "Terraform"
}

output "hello_response" {
  value = platform-api_hello.example.response
} 
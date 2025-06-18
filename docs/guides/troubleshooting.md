---
page_title: "Troubleshooting - AuthZed Provider"
description: |-
  Common issues and solutions when using the AuthZed Terraform provider.
---

# Troubleshooting

This guide covers common issues you might encounter when using the AuthZed Terraform provider and their solutions.

## Provider Installation Issues

### "Failed to query available provider packages" or "no available releases match"

If you encounter errors like:

```
Error: Failed to query available provider packages

Could not retrieve the list of available versions for provider authzed/authzed: 
no available releases match the given constraints
```

or:

```
Error: Failed to query available provider packages

Could not retrieve the list of available versions for provider authzed/authzed: 
the request failed after 2 attempts, please try again later
```

**Root Cause:**
This is often caused by locally cached provider files from development or previous installations. Terraform's "Implied Local Mirror Directories" feature finds these cached files and excludes the provider from direct registry installation, making newer versions appear unavailable.

**Solution:**

1. **Check for cached files:**
   ```bash
   find ~/.terraform.d -name "*authzed*"
   ```

2. **Remove cached files:**
   ```bash
   rm -rf ~/.terraform.d/plugins/*/authzed
   # or more specifically:
   rm -rf ~/.terraform.d/plugins/terraform.local/local/authzed
   rm -rf ~/.terraform.d/plugins/local.providers/local/authzed  
   rm -rf ~/.terraform.d/plugins/registry.terraform.io/authzed
   ```

3. **Reinitialize Terraform:**
   ```bash
   rm -rf .terraform .terraform.lock.hcl
   terraform init
   ```

**Prevention:**
- If you're switching from local development to published versions, always clear the local cache
- Consider using explicit `provider_installation` blocks in your `~/.terraformrc` file if you frequently switch between local and published providers

### Network Connectivity Issues

If you're behind a corporate firewall or using VPN, you might encounter timeout errors when Terraform tries to download providers.

**Solutions:**
- Check if your network can reach `registry.terraform.io` and `releases.hashicorp.com`
- If using a VPN, try toggling it on/off to see if it resolves connectivity issues
- Contact your network administrator about allowlisting HashiCorp's registry domains

### Version Constraints

Make sure your version constraints are correctly specified:

```hcl
terraform {
  required_providers {
    authzed = {
      source  = "authzed/authzed"
      version = "~> 0.1.0"  # Use appropriate version constraint
    }
  }
}
```

## Authentication Issues

### "Unauthorized" or "Authentication failed"

**Check your token:**
- Ensure your AuthZed Cloud API token is valid and not expired
- Verify the token has the necessary permissions for the resources you're managing
- Make sure the token is correctly set via environment variable or provider configuration

```bash
# Check if token is set
echo $AUTHZED_API_TOKEN
```

### "Invalid endpoint"

**Check your endpoint configuration:**
- Verify the endpoint URL is correct for your environment
- Ensure you're using the correct API version if specified

```hcl
provider "authzed" {
  endpoint    = "https://api.admin.stage.aws.authzed.net"  # Verify this URL
  token       = var.authzed_api_token
  api_version = "25r1"  # Optional, verify if needed
}
```

## Getting Help

If you continue to experience issues:

1. **Check the GitHub Issues:** Visit the [AuthZed Terraform Provider GitHub repository](https://github.com/authzed/terraform-provider-authzed/issues) to see if your issue has been reported or resolved.

2. **Enable Debug Logging:**
   ```bash
   export TF_LOG=DEBUG
   terraform init
   ```
   This will provide more detailed output to help diagnose issues.

3. **Contact Support:** If you're an AuthZed customer, contact your account team for assistance. 
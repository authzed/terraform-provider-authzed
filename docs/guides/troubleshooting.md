---
page_title: "Troubleshooting - AuthZed Provider"
description: |-
  Common issues and solutions when using the AuthZed Terraform provider.
---

# Troubleshooting

This guide covers common issues you might encounter when using the AuthZed Terraform provider and their solutions.

## ETag and Compression Issues

### Missing ETag Headers

**Problem**: You see errors like "API did not return the required ETag header" or resources fail to update properly.

**Solution**: The provider disables HTTP compression by default to preserve ETag headers reliably. This is a temporary workaround - do not attempt to re-enable compression as it will cause ETag header issues.

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

## Resource Management Issues

### "Provider produced inconsistent result after apply"

If you encounter errors like:

```
Error: Provider produced inconsistent result after apply

When applying changes to authzed_policy.example, provider 
"registry.terraform.io/authzed/authzed" produced an unexpected new value: 
.created_at: was "2025-01-08T10:00:00Z", but now cty.StringVal("").
```

**Root Cause:**
This error was caused by incorrect plan modifier configuration for computed fields in provider versions prior to v0.5.0.

## Performance and Parallelism

**Problem**: Terraform deployments fail with FGAM conflicts or service accounts disappear from state.

**Solution**: Use `parallelism=1`. This is a temporary workaround for Cloud API limitations that will be addressed in future provider versions.

### When to Use parallelism=1

```bash
terraform apply -parallelism=1
```

**Required for:**
- **Mixed resource types**: >8 total resources (prevents FGAM conflicts)
- **Service accounts**: >5 service accounts (resources disappear from Terraform state due to eventual consistency; wait logic added but `parallelism=1` still required)
- **Large deployments**: >50 resources (avoids API rate limits)

**Default parallelism works for:**
- Small deployments (≤8 resources)
- Single resource type deployments of roles or tokens (15+ resources)

**Performance Note:** While `parallelism=1` is inherently slower than concurrent execution, the provider implements several optimizations to minimize this impact:
- **Per-Permission System serialization lanes** that allow concurrent operations across different permission systems
- **Intelligent retry logic** with exponential backoff for API conflicts
- **Wait logic** to handle eventual consistency without unnecessary delays

These optimizations significantly reduce execution time compared to naive serial processing, though some performance trade-off remains necessary for reliability.

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
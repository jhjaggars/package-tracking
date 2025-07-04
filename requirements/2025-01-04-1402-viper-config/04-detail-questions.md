# Expert Detail Questions

## Q6: Should the Config interface pattern in internal/handlers/config_interface.go remain unchanged to maintain API compatibility?
**Default if unknown:** Yes (changing the interface would require updates to all handlers and could break external integrations)

## Q7: Will you require Viper's remote configuration capabilities (etcd, consul) for future cloud deployments?
**Default if unknown:** No (current deployment model is self-contained and remote config adds complexity)

## Q8: Should environment variable names keep their current format (USPS_API_KEY) or adopt Viper's nested convention (PKG_TRACKER_CARRIERS_USPS_API_KEY)?
**Default if unknown:** Yes (keep current names to avoid breaking existing deployments and documentation)

## Q9: Do you want Viper to automatically create missing configuration files with defaults when they don't exist?
**Default if unknown:** No (explicit configuration is safer than auto-generated files that might contain sensitive defaults)

## Q10: Should the email-tracker's --config flag continue to support only .env format or accept any Viper-supported format?
**Default if unknown:** No (accept any format - this provides deployment flexibility without breaking existing usage)
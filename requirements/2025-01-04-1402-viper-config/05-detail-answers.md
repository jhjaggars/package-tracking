# Expert Detail Answers

## Q6: Should the Config interface pattern in internal/handlers/config_interface.go remain unchanged to maintain API compatibility?
**Answer:** Yes

## Q7: Will you require Viper's remote configuration capabilities (etcd, consul) for future cloud deployments?
**Answer:** No

## Q8: Should environment variable names keep their current format (USPS_API_KEY) or adopt Viper's nested convention (PKG_TRACKER_CARRIERS_USPS_API_KEY)?
**Answer:** No, we should update documentation, this will simplify future configuration additions

## Q9: Do you want Viper to automatically create missing configuration files with defaults when they don't exist?
**Answer:** No, but we should add a default config example that users can copy and edit similar to the .env.example file

## Q10: Should the email-tracker's --config flag continue to support only .env format or accept any Viper-supported format?
**Answer:** No, accept a viper compatible format
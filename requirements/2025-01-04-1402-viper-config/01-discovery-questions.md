# Discovery Questions

## Q1: Should Viper configuration hot-reloading be enabled for automatic config updates without restarts?
**Default if unknown:** No (current system doesn't support hot-reloading, adding it could introduce unexpected behavior)

## Q2: Will you want to support multiple configuration file formats (YAML, TOML, JSON) or stick with current .env/JSON approach?
**Default if unknown:** Yes (Viper's multi-format support provides flexibility for different deployment scenarios)

## Q3: Should the existing CLI JSON config file (~/.package-tracker.json) be migrated to use Viper as well?
**Default if unknown:** Yes (consistency across all applications is beneficial)

## Q4: Do you need to maintain backward compatibility with existing .env files and environment variables?
**Default if unknown:** Yes (breaking existing deployments would be disruptive)

## Q5: Should configuration validation remain at the application level or be moved to Viper's validation features?
**Default if unknown:** No (keep validation at application level for better control and custom error messages)
package initconfig

// iterate over the parsed config and:
// - set missing default values, e.g., the default role of the user
func (*InitConfig) PostProcess() {
	// Todo check parsed config and apply defaults
}

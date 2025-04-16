package models

type PermissionsSystem struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	GlobalDnsPath string                 `json:"globalDnsPath"`
	SystemType    string                 `json:"systemType"`
	SystemState   PermissionsSystemState `json:"systemState"`
	Version       SystemVersion          `json:"version"`
}

type PermissionsSystemState struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type SystemVersion struct {
	CurrentVersion     SpiceDBVersion `json:"currentVersion"`
	HasUpdateAvailable bool           `json:"hasUpdateAvailable"`
	IsLockedToVersion  bool           `json:"isLockedToVersion"`
	OverrideImage      string         `json:"overrideImage"`
	SelectedChannel    string         `json:"selectedChannel"`
}

type SpiceDBVersion struct {
	DisplayName           string   `json:"displayName"`
	SupportedFeatureNames []string `json:"supportedFeatureNames"`
	Version               string   `json:"version"`
}

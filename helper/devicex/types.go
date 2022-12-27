package devicex

type devMemory struct {
	Total       uint64  `json:"total"`
	Available   uint64  `json:"available"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"usedPercent"`
	Free        uint64  `json:"free"`
}
type devCpu struct {
	Percent   float64 `json:"percent"`
	CoreCount int     `json:"coreCount"`
}
type devDisk struct {
	Name           string  `json:"name"`
	AvailableBytes uint64  `json:"availableBytes"`
	UsageBytes     uint64  `json:"usageBytes"`
	UsageRatio     float64 `json:"usageRatio"`
}
type devNet struct {
	InterfacesName string `json:"interfacesName"`
	Mac            string `json:"mac"`
	Ip             string `json:"ip"`
}

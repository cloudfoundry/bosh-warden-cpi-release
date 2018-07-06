package rpc

type DefaultContext struct {
	DirectorUUID string    `json:"director_uuid"`
	RequestID    string    `json:"request_id"`
	Vm           VmContext `json:"vm"`
}

type VmContext struct {
	Stemcell StemcellContext `json:"stemcell"`
}

type StemcellContext struct {
	ApiVersion int `json:"api_version"`
}

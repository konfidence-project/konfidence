package handler

type StageCondition struct {
	Type               string `json:"type"`
	Status             string `json:"status"`
	Reason             string `json:"reason"`
	Message            string `json:"message"`
	LastTransitionTime string `json:"lastTransitionTime"`
}

type StageResponse struct {
	Name       string           `json:"name"`
	Namespace  string           `json:"namespace"`
	Vector     string           `json:"vector"`
	Conditions []StageCondition `json:"conditions"`
}

type StageListResponse struct {
	Stages []StageResponse `json:"stages"`
}

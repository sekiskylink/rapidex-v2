package observability

type TraceReference struct {
	RequestUID  string `json:"requestUid"`
	DeliveryUID string `json:"deliveryUid"`
	JobUID      string `json:"jobUid"`
}

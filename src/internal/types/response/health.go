package response

// HealthResponse 健康检查响应
type HealthResponse struct {
    Status  string `json:"status"`
    Details string `json:"details,omitempty"`
}

// ReadinessResponse 就绪检查响应
type ReadinessResponse struct {
    Status string            `json:"status"`
    Reason string            `json:"reason,omitempty"`
    Checks map[string]string `json:"checks,omitempty"`
}

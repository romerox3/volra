package output

// Error codes for doctor command (E1xx).
const (
	CodeDockerNotInstalled  = "E101"
	CodeDockerNotRunning    = "E102"
	CodeComposeNotAvailable = "E103"
	CodePortInUse           = "E104"
	CodePythonNotFound      = "E105"
	CodeInsufficientDisk    = "E106"
)

// Error codes for setup command (E2xx).
const (
	CodeNoPythonProject    = "E201"
	CodeNoEntryPoint       = "E202"
	CodeAgentfileExists    = "E203"
	CodeSetupReserved4     = "E204"
	CodeSetupReserved5     = "E205"
	CodeSetupReserved6     = "E206"
)

// Error codes for deploy command (E3xx).
const (
	CodeDeployDockerNotRunning = "E301"
	CodeBuildFailed            = "E302"
	CodeHealthCheckFailed      = "E303"
	CodeOOMKilled              = "E304"
	CodeEnvNotFound            = "E305"
	CodeDeployReserved6        = "E306"
	CodeGPUNotAvailable        = "E307"
	CodeGPUCheckFailed         = "E308"
)

// Error codes for status command (E4xx).
const (
	CodeNoDeployment         = "E401"
	CodeStatusDockerNotRunning = "E402"
)

// Error codes for shared/agentfile (E5xx).
const (
	CodeInvalidAgentfile       = "E501"
	CodeUnsupportedVersion     = "E502"
)

// Error codes for lifecycle commands (E6xx).
const (
	CodeNoDeploymentFound    = "E601"
	CodeComposeWatchRequired = "E602"
)

// Warning codes for lifecycle commands (W6xx).
const (
	CodeWarnComposeWatchOld = "W601"
)

// Error codes for eval command (E7xx).
const (
	CodeNoBaseline            = "E701"
	CodePrometheusUnreachable = "E702"
	CodeInvalidEvalConfig     = "E703"
	CodePromQLFailed          = "E704"
	CodeEvalRegression        = "E705"
)

// Error codes for hub command (E8xx).
const (
	CodeNoAgentsRegistered = "E801"
	CodeHubAlreadyRunning  = "E802"
)

// Error codes for gateway command (E9xx).
const (
	CodeGatewayNoAgents     = "E901"
	CodeGatewaySpawnFailed  = "E902"
	CodeGatewayInitFailed   = "E903"
	CodeGatewayToolsFailed  = "E904"
	CodeGatewayAgentTimeout = "E905"
)

// Error codes for marketplace command (E12xx).
const (
	CodeMarketplaceFetch    = "E1201"
	CodeMarketplaceNotFound = "E1202"
)

// Error codes for compliance command (E13xx).
const (
	CodeNoAgentfileForCompliance = "E1301"
)

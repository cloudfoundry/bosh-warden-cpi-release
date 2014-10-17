package vm

type AgentEnvServiceFactory interface {
	New(WardenFileService, string) AgentEnvService
}

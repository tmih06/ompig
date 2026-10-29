package coding

// Ports packages/coding-agent/src/core/client-identity.ts.

import (
	"github.com/tmih06/ompig/ai"
	icodingagent "github.com/tmih06/ompig/internal/codingagent"
)

// getClientIdentity is how Ompig's requests identify it to providers: a detailed `User-Agent` with Ompig's version, and attribution to Ompig as a CLI agent while install telemetry is enabled.
func getClientIdentity(settings *icodingagent.SettingsManager) *ai.ClientIdentity {
	identity := &ai.ClientIdentity{UserAgent: new(ai.UserAgentDetailed), Version: new(Version)}
	if settings.IsInstallTelemetryEnabled() {
		identity.Attribution = new(ai.PiAttribution)
	} else {
		identity.AttributionNull = true
	}
	return identity
}

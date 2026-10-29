// Package pipelinediff compares the conversation Ompig puts on the wire with the one Pi 1.1.0 puts there.
//
// Both sides receive the same seeded random AgentMessage history (user, assistant, tool result, bash execution, custom, branch summary, compaction summary and system messages; aborted and errored turns; parallel calls; partial results). Pi runs coding-agent's convertToLlm, then the API's stream, which runs transformMessages and the converter; the request lands on a local HTTP server. Ompig runs agent.ConvertToLLM(agent.NormalizeMessages(...)), then the same API. The members of the request body that carry the conversation must be equal.
//
// Pi is the oracle: the comparison never reads Ompig's output to decide what is right. Where Pi's own request breaks the provider's message contract (internal/requestvalidity), Ompig's request must still meet the contract, because Ompig repairs such histories (D48, late-result recovery).
//
// OMPIG_PIPELINE_CASES sets the cases per target and mode (12 by default, 150 when CI is set; make test-pipeline-fuzz runs 1000) and OMPIG_PIPELINE_SEED the first seed. A failure prints the smallest history that still fails; OMPIG_PIPELINE_CASE_FILE=<file> with TestShow prints both requests for it.
package pipelinediff

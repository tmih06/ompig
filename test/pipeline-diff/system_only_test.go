package pipelinediff

import "testing"

// A history of system messages only, or of one user message without content (empty.test.ts testEmptyMessage), has no `contents` for Google. Pi's @google/genai SDK throws "contents are required" before sending; Ompig must fail before sending with the same message instead of putting a request on the wire that Gemini rejects.
func TestSystemOnlyHistoryFailsBeforeSendingLikePi(t *testing.T) {
	h := newHarness(t)
	histories := map[string][]message{
		"system only":        {{"role": "system", "content": "only a system message", "timestamp": 1700000000000}},
		"empty user content": {{"role": "user", "content": []any{}, "timestamp": 1700000000000}},
	}
	for name, history := range histories {
		for _, tgt := range targets {
			if tgt.API != "google-generative-ai" {
				continue
			}
			c := testCase{Mode: Chaos, Seed: 1, Target: tgt, Messages: history}
			if o := h.judge(c); o.Kind != "equal" || o.Detail != "both fail before sending" {
				t.Fatalf("%s, %s: %s %s", name, tgt.Name, o.Kind, o.Detail)
			}
			piURL, piKey := h.capture.slot("pi")
			piMessage, err := h.pi.runWithStatus(c, piURL)
			if err != nil {
				t.Fatal(err)
			}
			ompigURL, ompigKey := h.capture.slot("ompig")
			_, ompigMessage, err := ompigRun(c, ompigURL, true)
			if err != nil {
				ompigMessage = err.Error()
			}
			_, piSent := h.capture.take(piKey)
			_, ompigSent := h.capture.take(ompigKey)
			if piSent || ompigSent || piMessage != "contents are required" || ompigMessage != piMessage {
				t.Fatalf("%s, %s: pi sent %v error %q, ompig sent %v error %q", name, tgt.Name, piSent, piMessage, ompigSent, ompigMessage)
			}
		}
	}
}

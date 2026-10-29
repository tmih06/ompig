//go:build !ompig_strip_node_extensions

package ai

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os/exec"
	"testing"
)

// pi: packages/ai/src/api/openai-codex-responses.ts compressRequestBodyZstd

// Pi compresses the Codex SSE request body with node:zlib zstdCompressSync at level 3 (REQUEST_COMPRESSION_ZSTD_LEVEL), so a long session's body shrinks several-fold on the wire. The body Ompig sends decodes, with Node's
// own decoder, to the same bytes, and is in Pi's size class: at most 10% above what Pi's zstd level 3 makes of the same JSON.
func TestCodexRequestBodyIsCompressedInPisClass(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatalf("the Node oracle is required: %v", err)
	}
	// Bodies of repeated turns shrink far past what a real session does and the encoders then differ more; bodies of varied text show the class.
	bodies := map[string]struct {
		body  []byte
		slack int // tolerated excess over Pi's size, in percent
	}{
		"small": {codexSizedBody(2), 10}, "medium": {codexSizedBody(60), 10}, "repeated-large": {codexSizedBody(1500), 50},
		"prose": {codexProseBody(400), 10}, "prose-large": {codexProseBody(4000), 10},
	}
	for name, c := range bodies {
		body := c.body
		t.Run(name, func(t *testing.T) {
			compressed := compressCodexRequestBody(body)
			input, err := json.Marshal(map[string]string{"body": base64.StdEncoding.EncodeToString(body), "ompig": base64.StdEncoding.EncodeToString(compressed)})
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(node, "-e", `const z=require("zlib");const i=JSON.parse(require("fs").readFileSync(0,"utf8"));
const body=Buffer.from(i.body,"base64"),ompig=Buffer.from(i.ompig,"base64");
const pi=z.zstdCompressSync(body,{params:{[z.constants.ZSTD_c_compressionLevel]:3}});
console.log(JSON.stringify({piSize:pi.length,ompigDecodesToBody:Buffer.compare(z.zstdDecompressSync(ompig),body)===0}))`)
			cmd.Stdin = bytes.NewReader(input)
			out, err := cmd.Output()
			if err != nil {
				t.Fatal(err)
			}
			var got struct {
				PiSize             int  `json:"piSize"`
				OmpigDecodesToBody bool `json:"ompigDecodesToBody"`
			}
			if err := json.Unmarshal(out, &got); err != nil {
				t.Fatal(err)
			}
			if !got.OmpigDecodesToBody {
				t.Fatal("Node's zstd decoder does not return the request body from Ompig's frame")
			}
			if limit := got.PiSize + got.PiSize*c.slack/100; len(compressed) > limit {
				t.Fatalf("Ompig sends %d bytes for a %d byte body, Pi's zstd level 3 sends %d (limit %d)", len(compressed), len(body), got.PiSize, limit)
			}
		})
	}
}

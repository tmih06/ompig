// Pi's InMemoryTelemetryContext as the other end of a carrier exchange with Ompig: it prints the carrier of one of its spans and continues each carrier Ompig injected.
import { readFileSync, realpathSync } from "node:fs";
import { pathToFileURL } from "node:url";

const root = realpathSync(new URL("../../extensions/sdk-ts/node_modules/@earendil-works/pi-coding-agent/node_modules/@earendil-works/pi-telemetry/", import.meta.url)) + "/";
if (JSON.parse(readFileSync(root + "package.json", "utf8")).version !== process.argv[2]) throw new Error("Wrong Pi oracle version");
const { InMemoryTelemetryContext } = await import(pathToFileURL(root + "dist/memory.js"));
let input = "";
process.stdin.setEncoding("utf8");
for await (const chunk of process.stdin) input += chunk;

const telemetry = new InMemoryTelemetryContext();
let piCarrier;
await telemetry.startSpan({ name: "pi-sender" }, (span) => { piCarrier = telemetry.inject(span); });
for (const [index, carrier] of JSON.parse(input).entries()) await telemetry.extract(carrier).startSpan({ name: `from-ompig-${index}` }, () => {});
process.stdout.write(JSON.stringify({ piCarrier, spans: telemetry.getSpans() }));

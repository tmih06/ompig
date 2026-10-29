// Ompig: the azure-openai-responses implementation runs in Ompig's host (D74); see automation/gen/vendor-pi-dist.sh.
import { bridgeApi } from "../../../pi-ai-bridge.mjs";
export const { stream, streamSimple } = bridgeApi("azure-openai-responses");

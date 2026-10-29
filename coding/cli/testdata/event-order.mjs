// event-order.mjs appends each lifecycle event it receives to the file named
// by OMPIG_TEST_EVENT_LOG.
import { appendFileSync } from "node:fs";

export default function eventOrderExtension(pi) {
  for (const event of ["agent_start", "message_end", "turn_end", "agent_end", "session_shutdown"]) {
    pi.on(event, async () => {
      appendFileSync(process.env.OMPIG_TEST_EVENT_LOG, `${event}\n`);
    });
  }
}

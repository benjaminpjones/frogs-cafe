import type { Reporter, File, Task } from "vitest";

function statusTag(task: Task): string {
  if (task.mode === "todo") return "\x1b[33m[TODO]\x1b[0m";
  if (task.mode === "skip") return "\x1b[36m[SKIP]\x1b[0m";
  if (task.result?.state === "pass") return "\x1b[32m[PASS]\x1b[0m";
  if (task.result?.state === "fail") return "\x1b[31m[FAIL]\x1b[0m";
  return "[????]";
}

function getTests(task: Task): Task[] {
  if (task.type === "test") return [task];
  if (task.type === "suite" && task.tasks) {
    return task.tasks.flatMap(getTests);
  }
  return [];
}

export default class ComplianceReporter implements Reporter {
  onFinished(files?: File[]) {
    if (!files) return;

    const target = process.env["BIK_TARGET"] ?? "(unknown)";
    console.log();
    console.log("BIK Protocol Compliance Report");
    console.log(`Target: ${target}`);
    console.log(`Date:   ${new Date().toISOString().split("T")[0]}`);
    console.log();

    let passed = 0;
    let failed = 0;
    let skipped = 0;
    let todo = 0;

    for (const file of files) {
      for (const suite of file.tasks) {
        if (suite.type === "suite") {
          console.log(suite.name);

          const tests = getTests(suite);
          for (const test of tests) {
            const tag = statusTag(test);
            console.log(`  ${tag} ${test.name}`);

            if (test.result?.state === "fail" && test.result.errors) {
              for (const err of test.result.errors) {
                const msg = err.message?.split("\n")[0] ?? "unknown error";
                console.log(`         ${msg}`);
              }
            }

            if (test.mode === "todo") todo++;
            else if (test.mode === "skip") skipped++;
            else if (test.result?.state === "pass") passed++;
            else if (test.result?.state === "fail") failed++;
          }
          console.log();
        }
      }
    }

    const total = passed + failed + skipped + todo;
    console.log(
      `Summary: ${passed} passed, ${failed} failed, ${skipped} skipped, ${todo} todo (${total} total)`,
    );
    console.log();
  }
}

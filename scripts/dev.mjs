import { spawn } from "node:child_process";

const npm = process.platform === "win32" ? "npm.cmd" : "npm";
const deno = process.platform === "win32" ? "deno.exe" : "deno";
const env = { ...process.env, APP_ENV: process.env.APP_ENV ?? "local" };
const children = [
  spawn(deno, ["task", "dev"], { stdio: "inherit", env }),
  spawn(npm, ["run", "dev:web"], { stdio: "inherit", env }),
];

let stopping = false;

function stop(exitCode = 0) {
  if (stopping) return;
  stopping = true;
  for (const child of children) child.kill("SIGTERM");
  process.exitCode = exitCode;
}

for (const child of children) {
  child.on("error", () => stop(1));
  child.on("exit", (code, signal) => {
    if (!stopping) stop(code ?? (signal ? 1 : 0));
  });
}

process.on("SIGINT", () => stop(0));
process.on("SIGTERM", () => stop(0));

#!/usr/bin/env node
"use strict";
const path = require("path");
const { spawnSync } = require("child_process");

const name = process.platform === "win32" ? "devrounds.exe" : "devrounds";
const exe = path.join(__dirname, "..", "vendor", name);
const res = spawnSync(exe, process.argv.slice(2), { stdio: "inherit" });
if (res.error) {
  console.error("devrounds: could not run " + exe + ": " + res.error.message);
  process.exit(1);
}
process.exit(res.status === null ? 1 : res.status);

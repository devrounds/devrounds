"use strict";
const fs = require("fs");
const os = require("os");
const path = require("path");
const crypto = require("crypto");
const { execFileSync } = require("child_process");

const version = require("./package.json").version;
const platforms = { linux: "linux", darwin: "darwin", win32: "windows" };
const arches = { x64: "amd64", arm64: "arm64" };

const goos = platforms[process.platform];
const goarch = arches[process.arch];
if (!goos || !goarch) {
  console.error("devrounds: unsupported platform " + process.platform + "/" + process.arch);
  process.exit(1);
}

const ext = goos === "windows" ? "zip" : "tar.gz";
const archive = "devrounds_" + version + "_" + goos + "_" + goarch + "." + ext;
const base = "https://github.com/devrounds/devrounds/releases/download/v" + version;
const vendor = path.join(__dirname, "vendor");
const exe = path.join(vendor, goos === "windows" ? "devrounds.exe" : "devrounds");

async function download(url) {
  const res = await fetch(url);
  if (!res.ok) {
    throw new Error("download failed (" + res.status + "): " + url);
  }
  return Buffer.from(await res.arrayBuffer());
}

function tarCommand() {
  if (process.platform === "win32") {
    return path.join(process.env.SystemRoot || "C:\\Windows", "System32", "tar.exe");
  }
  return "tar";
}

async function main() {
  if (fs.existsSync(exe)) {
    return;
  }
  const data = await download(base + "/" + archive);
  const sums = (await download(base + "/checksums.txt")).toString("utf8");
  const line = sums.split(/\r?\n/).find((l) => l.trim().endsWith(archive));
  if (!line) {
    throw new Error("no checksum listed for " + archive);
  }
  const expected = line.trim().split(/\s+/)[0];
  const actual = crypto.createHash("sha256").update(data).digest("hex");
  if (expected !== actual) {
    throw new Error("checksum mismatch for " + archive);
  }
  fs.mkdirSync(vendor, { recursive: true });
  const tmp = path.join(os.tmpdir(), archive + "." + process.pid);
  fs.writeFileSync(tmp, data);
  try {
    execFileSync(tarCommand(), ["-xf", tmp, "-C", vendor], { stdio: "inherit" });
  } finally {
    fs.rmSync(tmp, { force: true });
  }
  if (goos !== "windows") {
    fs.chmodSync(exe, 0o755);
  }
}

main().catch((err) => {
  console.error("devrounds: " + err.message);
  process.exit(1);
});

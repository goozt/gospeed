#!/usr/bin/env node
// Postinstall: fetch the matching prebuilt archive from the GitHub Release
// matching this package's version, extract just the `gospeed-server` binary,
// and write a JS shim so `npm i -g @goozt/gospeed-server` produces a working
// `gospeed-server` command.

const fs = require("fs");
const path = require("path");
const https = require("https");
const { execSync } = require("child_process");

const OWNER = "goozt";
const REPO = "gospeed";
const BINARIES = ["gospeed-server"];
const PROJECT_NAME = "gospeed";

const pkg = require("./package.json");
const VERSION = "v" + pkg.version;

const PLATFORM = (() => {
  switch (process.platform) {
    case "darwin":
      return "darwin";
    case "linux":
      return "linux";
    case "win32":
      return "windows";
    case "freebsd":
      return "freebsd";
    default:
      throw new Error(`unsupported platform: ${process.platform}`);
  }
})();

const ARCH = (() => {
  switch (process.arch) {
    case "x64":
      return "amd64";
    case "arm64":
      return "arm64";
    case "ia32":
      return "386";
    case "arm":
      return "arm";
    case "s390x":
      return "s390x";
    case "ppc64":
      return "ppc64le";
    default:
      throw new Error(`unsupported arch: ${process.arch}`);
  }
})();

const EXT = PLATFORM === "windows" ? "zip" : "tar.gz";
const ARCHIVE = `${PROJECT_NAME}_${pkg.version}_${PLATFORM}_${ARCH}.${EXT}`;
const URL = `https://github.com/${OWNER}/${REPO}/releases/download/${VERSION}/${ARCHIVE}`;

const binDir = path.join(__dirname, "bin");
fs.mkdirSync(binDir, { recursive: true });

const archivePath = path.join(binDir, ARCHIVE);

function get(url, dest) {
  return new Promise((resolve, reject) => {
    https
      .get(url, { headers: { "User-Agent": "npm-postinstall" } }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          return get(res.headers.location, dest).then(resolve, reject);
        }
        if (res.statusCode !== 200) {
          return reject(new Error(`HTTP ${res.statusCode} for ${url}`));
        }
        const f = fs.createWriteStream(dest);
        res.pipe(f);
        f.on("finish", () => f.close(resolve));
        f.on("error", reject);
      })
      .on("error", reject);
  });
}

async function main() {
  console.log(`Downloading ${URL}`);
  await get(URL, archivePath);

  if (EXT === "tar.gz") {
    execSync(`tar -xzf "${archivePath}" -C "${binDir}"`, { stdio: "inherit" });
  } else {
    execSync(
      `powershell -NoLogo -NoProfile -Command "Expand-Archive -Force -Path '${archivePath}' -DestinationPath '${binDir}'"`,
      { stdio: "inherit" },
    );
  }

  fs.unlinkSync(archivePath);

  const keep = new Set(
    BINARIES.flatMap((b) => [b, `${b}.exe`]),
  );
  for (const entry of fs.readdirSync(binDir)) {
    const full = path.join(binDir, entry);
    if (fs.statSync(full).isFile() && /^[a-z0-9-]+(\.exe)?$/.test(entry) && !keep.has(entry)) {
      fs.unlinkSync(full);
    }
  }

  for (const binary of BINARIES) {
    const binName = PLATFORM === "windows" ? `${binary}.exe` : binary;
    const shimPath = path.join(binDir, `${binary}.js`);
    const shim = `#!/usr/bin/env node
const { spawn } = require("child_process");
const path = require("path");
const child = spawn(path.join(__dirname, "${binName}"), process.argv.slice(2), { stdio: "inherit" });
child.on("exit", (code) => process.exit(code === null ? 1 : code));
`;
    fs.writeFileSync(shimPath, shim, { mode: 0o755 });

    if (PLATFORM !== "windows") {
      fs.chmodSync(path.join(binDir, binName), 0o755);
    }
  }

  console.log(`Installed ${BINARIES.join(", ")} for ${PLATFORM}/${ARCH}`);
}

main().catch((err) => {
  console.error(err.message);
  process.exit(1);
});

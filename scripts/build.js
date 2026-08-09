import { copyFileSync, existsSync, lstatSync, mkdirSync, readdirSync, rmSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";

const root = resolve(import.meta.dirname, "..");
const output = resolve(root, "dist");
const client = resolve(output, "client");
const sources = ["index.html", "css", "js", "data", "assets"];

rmSync(output, { recursive: true, force: true });
mkdirSync(client, { recursive: true });

function copySource(from, to) {
  if (lstatSync(from).isDirectory()) {
    mkdirSync(to, { recursive: true });
    for (const entry of readdirSync(from)) copySource(resolve(from, entry), resolve(to, entry));
    return;
  }
  copyFileSync(from, to);
}

for (const source of sources) {
  const from = resolve(root, source);
  if (existsSync(from)) copySource(from, resolve(client, source));
}

const worker = `export default {
  async fetch(request, env) {
    const url = new URL(request.url);
    if (url.pathname === "/") url.pathname = "/index.html";
    return env.ASSETS.fetch(new Request(url, request));
  }
};
`;

const workerPath = resolve(output, "server", "index.js");
mkdirSync(dirname(workerPath), { recursive: true });
writeFileSync(workerPath, worker, "utf8");

console.log("已生成可发布的静态站点。");

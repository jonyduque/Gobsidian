#!/usr/bin/env node
/**
 * gobsidian-installer
 * ---------------------------------------------------------------------------
 * Instala o gobsidian (servidor MCP para cofres Obsidian) e o registra nos
 * hosts de IA presentes na maquina. Porte para Node.js do install.ps1
 * original (Windows-only) -- cross-platform: Windows, macOS e Linux.
 *
 * SEM DEPENDENCIAS NPM DE PROPOSITO: este arquivo e pensado para ser baixado
 * e executado direto com `node install.js`, a partir de um one-liner bash ou
 * PowerShell (curl/irm + node), sem passo de `npm install` no meio.
 *
 * O gobsidian serve UM cofre por processo: `--vault` e uma raiz so, e o
 * confinamento de caminho de toda tool e verificado contra ela. Varios cofres
 * viram varias entradas de MCP, cada uma com seu nome, apontando para o mesmo
 * binario. Este instalador cuida disso: escolha quantos cofres quiser e ele
 * registra um servidor por cofre.
 *
 * Se o gobsidian ja estiver instalado e na versao da release mais recente, ele
 * PULA o download e vai direto para a configuracao de cofres.
 *
 * Uso tipico:
 *
 *   node install.js
 *   node install.js --vault "/caminho/do/cofre" --yes
 *   node install.js --vault "/cofre/um" --vault "/cofre/dois"
 *   node install.js --hosts claude-desktop,claude-code --read-only
 *
 * Variaveis de ambiente equivalentes as flags (uteis em CI/scripts):
 *   GOBSIDIAN_VAULT, GOBSIDIAN_VERSION, GOBSIDIAN_INSTALL_DIR
 *   GITHUB_TOKEN ou GH_TOKEN -- para repositorio privado / rate limit
 *
 * Flags:
 *   --vault <path>        Raiz do cofre. REPETIVEL, ou separada por virgula.
 *   --name <nome>         Nome do servidor MCP. So com um cofre; com varios,
 *                         o nome sai do nome da pasta de cada um.
 *   --version <tag>       Tag da release (padrao: mais recente).
 *   --install-dir <path>  Onde por o binario.
 *   --hosts <a,b,c>       Chaves de host a configurar, sem perguntar.
 *   --yes                 Nao pergunta nada. Exige --vault.
 *   --read-only           Registra o servidor com --read-only nos hosts.
 *   --no-path             Nao mexe no PATH.
 *   --force               Reinstala o binario mesmo se a versao ja for a atual.
 * ---------------------------------------------------------------------------
 */

import fs from "node:fs";
import fsp from "node:fs/promises";
import path from "node:path";
import os from "node:os";
import crypto from "node:crypto";
import { execFileSync, spawnSync } from "node:child_process";
import { createInterface } from "node:readline/promises";

const REPO = "jonyduque/Gobsidian";
const SERVER_KEY = "gobsidian";
const PLATFORM = process.platform; // 'win32' | 'darwin' | 'linux'
const IS_WIN = PLATFORM === "win32";
const EXE_NAME = IS_WIN ? "gobsidian.exe" : "gobsidian";

// -----------------------------------------------------------------------
// Log helpers -- ANSI, sem depender de chalk/consola (script standalone:
// precisa rodar so com `node`, sem node_modules).
//
// Marcadores em ASCII puro, os MESMOS do install.ps1 da raiz ([OK], [i],
// [!], [...]). Emoji aqui renderiza como lixo num console PowerShell em
// CP-850, e o instalador e o primeiro programa que alguem roda -- antes de
// ter como saber se o que ele esta vendo e um defeito da ferramenta ou da
// fonte do terminal. A mensagem "instale o Node" cai justamente sobre quem
// ja esta com problema.
// -----------------------------------------------------------------------
const ANSI = {
  reset: "\x1b[0m", bold: "\x1b[1m",
  cyan: "\x1b[36m", green: "\x1b[32m", gray: "\x1b[90m", yellow: "\x1b[33m", red: "\x1b[31m",
};
const useColor = process.stdout.isTTY && !process.env.NO_COLOR;
function paint(code, s) { return useColor ? `${code}${s}${ANSI.reset}` : s; }

const log = {
  title: (m) => console.log(paint(ANSI.bold + ANSI.cyan, `\n  ${m}\n`)),
  step: (m) => console.log(paint(ANSI.cyan, `[...] ${m}`)),
  ok: (m) => console.log(paint(ANSI.green, `[OK] ${m}`)),
  info: (m) => console.log(paint(ANSI.gray, `[i] ${m}`)),
  warn: (m) => console.log(paint(ANSI.yellow, `[!] ${m}`)),
  err: (m) => console.log(paint(ANSI.red, `[!] ${m}`)),
};

// -----------------------------------------------------------------------
// Prompts interativos (numerados, ao estilo do install.ps1 original) --
// so via readline, sem lib de terceiros.
// -----------------------------------------------------------------------
async function ask(question) {
  const rl = createInterface({ input: process.stdin, output: process.stdout });
  try {
    return (await rl.question(question)).trim();
  } finally {
    rl.close();
  }
}

// Escolha de UM OU VARIOS cofres. Devolve uma lista de caminhos validados.
//
// Um prompt so, que aceita numeros do menu, "t" para todos, ou caminhos
// colados; e que REPETE quando a resposta nao serve. A versao anterior tinha
// dois prompts em sequencia -- o do menu e, se ele nao desse numero valido, um
// pedindo caminho -- e numa execucao real o primeiro voltou vazio: o "1" caiu no
// segundo, que o leu como caminho e reprovou com "Cofre nao encontrado: 1".
async function promptVaults(encontrados) {
  for (let tentativa = 1; tentativa <= 3; tentativa++) {
    console.log("");
    if (encontrados.length) {
      console.log(paint(ANSI.cyan, "  Cofres que o Obsidian conhece:"));
      encontrados.forEach((v, i) => console.log(`   [${i + 1}] ${v}`));
      console.log("");
      console.log(paint(ANSI.gray, "  Numeros separados por virgula para varios, 't' para todos,"));
      console.log(paint(ANSI.gray, "  ou cole um ou mais caminhos separados por virgula."));
    } else {
      console.log(paint(ANSI.gray, "  Nenhum cofre registrado pelo Obsidian foi encontrado."));
      console.log(paint(ANSI.gray, "  Cole o caminho do cofre (varios, separados por virgula)."));
    }

    const r = (await ask("  Cofre(s): ")).trim();
    if (!r) { log.warn("Resposta vazia."); continue; }
    if (r.toLowerCase() === "t" && encontrados.length) return encontrados.slice();

    const escolhidos = [];
    let erro = null;
    for (const parte of splitVaults(r)) {
      if (/^\d+$/.test(parte)) {
        const n = Number(parte);
        if (n >= 1 && n <= encontrados.length) escolhidos.push(encontrados[n - 1]);
        else { erro = `Nao ha opcao ${parte}. O menu vai de 1 a ${encontrados.length}.`; break; }
      } else if (fs.existsSync(parte) && fs.statSync(parte).isDirectory()) {
        escolhidos.push(parte);
      } else {
        erro = `Nao existe um diretorio em: ${parte}`; break;
      }
    }
    if (erro) { log.warn(erro); continue; }
    if (escolhidos.length) return [...new Set(escolhidos)];
    log.warn("Nada selecionado.");
  }
  return [];
}

// Retorna a lista de chaves (subconjunto de detectados) escolhida pelo usuario.
async function promptHostChoice(detectados) {
  console.log("");
  console.log(paint(ANSI.cyan, "  Em quais registrar o gobsidian?"));
  detectados.forEach((h, i) => console.log(`   [${i + 1}] ${h.nome}`));
  console.log("   [t] todos");
  console.log("   [n] nenhum");
  console.log("");
  console.log(paint(ANSI.gray, "  Numeros separados por virgula, ou t/n."));
  const r = (await ask("  Escolha: ")).toLowerCase();
  if (r === "t" || r === "") return detectados;
  if (r === "n") return [];
  const escolhidos = [];
  for (const p of r.split(/[,\s]+/).filter(Boolean)) {
    const n = Number(p);
    if (Number.isInteger(n) && n >= 1 && n <= detectados.length) escolhidos.push(detectados[n - 1]);
    else log.warn(`ignorando entrada invalida: ${p}`);
  }
  return escolhidos;
}

// -----------------------------------------------------------------------
// 0. Argumentos e ambiente
// -----------------------------------------------------------------------
function parseArgs(argv) {
  const opts = {
    // Sempre lista, mesmo com um cofre so: um caminho especial para "um" e
    // outro para "varios" seria duas implementacoes do mesmo fluxo, e a de um
    // so e a que quase nunca e exercitada depois.
    vaults: splitVaults(process.env.GOBSIDIAN_VAULT),
    name: null,
    version: process.env.GOBSIDIAN_VERSION || null,
    installDir: process.env.GOBSIDIAN_INSTALL_DIR || null,
    hosts: null,
    yes: false,
    readOnly: false,
    noPath: false,
    force: false,
  };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    const next = () => argv[++i];
    switch (a) {
      case "--vault": opts.vaults.push(...splitVaults(next())); break;
      case "--name": opts.name = next(); break;
      case "--version": opts.version = next(); break;
      case "--install-dir": opts.installDir = next(); break;
      case "--hosts": opts.hosts = next().split(",").map((s) => s.trim()).filter(Boolean); break;
      case "--yes": case "-y": opts.yes = true; break;
      case "--read-only": opts.readOnly = true; break;
      case "--no-path": opts.noPath = true; break;
      case "--force": opts.force = true; break;
      case "--help": case "-h": printHelp(); process.exit(0); break;
      default:
        log.warn(`argumento desconhecido, ignorando: ${a}`);
    }
  }
  return opts;
}

// Aceita "--vault A --vault B" e "--vault A,B". A virgula NAO e separador em
// Windows (C: nao usa virgula), mas e em caminho nenhum dos tres sistemas, e
// digitar a lista de uma vez e o que alguem tenta primeiro.
function splitVaults(s) {
  if (!s) return [];
  return s.split(",").map((x) => x.trim().replace(/^["']|["']$/g, "")).filter(Boolean);
}

// Nome do servidor MCP a partir do caminho do cofre.
//
// Com UM cofre o nome e "gobsidian", que e o que a documentacao mostra e o que
// quem tem um cofre so espera ver. Com varios, cada um precisa de nome proprio,
// e o nome da pasta e a unica coisa que o usuario reconhece de imediato.
function serverKeyForVault(vaultPath, total, override) {
  if (override) return override;
  if (total <= 1) return SERVER_KEY;
  const base = path.basename(vaultPath)
    .normalize("NFD").replace(/[̀-ͯ]/g, "")   // tira acento
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
  return base ? `${SERVER_KEY}-${base}` : SERVER_KEY;
}

function printHelp() {
  console.log(fs.readFileSync(new URL(import.meta.url), "utf8").split("*/")[0]);
}

function isInteractive(opts) {
  if (opts.yes) return false;
  return Boolean(process.stdin.isTTY) && Boolean(process.stdout.isTTY);
}

function isAdmin() {
  try {
    if (IS_WIN) {
      // 'net session' so retorna com sucesso com privilegios elevados.
      execFileSync("net", ["session"], { stdio: "ignore" });
      return true;
    }
    return typeof process.getuid === "function" && process.getuid() === 0;
  } catch {
    return false;
  }
}

function testExe(name) {
  const probe = IS_WIN ? "where" : "which";
  const r = spawnSync(probe, [name], { stdio: "ignore", shell: false });
  return r.status === 0;
}

// -----------------------------------------------------------------------
// 0b. Instalacao ja existente -- versao e localizacao
// -----------------------------------------------------------------------

// Onde ja ha um gobsidian: primeiro o diretorio em que ESTE instalador poria,
// depois o PATH. A ordem importa: se as duas coisas existirem, a que o
// instalador atualiza e a dele, e reportar a do PATH daria a impressao de ter
// atualizado um binario que ficou para tras.
function findInstalledExe(installDir) {
  const candidatos = [path.join(installDir, EXE_NAME)];
  const probe = IS_WIN ? "where" : "which";
  const r = spawnSync(probe, [IS_WIN ? "gobsidian" : "gobsidian"], { encoding: "utf8" });
  if (r.status === 0 && r.stdout) {
    for (const linha of r.stdout.split(/\r?\n/).map((s) => s.trim()).filter(Boolean)) {
      candidatos.push(linha);
    }
  }
  for (const c of candidatos) {
    try { if (fs.existsSync(c) && fs.statSync(c).isFile()) return c; } catch { /* segue */ }
  }
  return null;
}

// A saida de `gobsidian version` e "gobsidian v1.0.0 (sha) data". Interessa a
// tag, que e o que a release nomeia.
function readInstalledVersion(exe) {
  try {
    const out = execFileSync(exe, ["version"], { encoding: "utf8", timeout: 10000 }).trim();
    const m = out.match(/\bv\d+\.\d+\.\d+\S*/);
    return { raw: out, tag: m ? m[0] : null };
  } catch {
    return { raw: null, tag: null };
  }
}

// -----------------------------------------------------------------------
// 1. Qual versao / release do GitHub
// -----------------------------------------------------------------------
function githubHeaders() {
  const token = process.env.GITHUB_TOKEN || process.env.GH_TOKEN;
  const headers = { "User-Agent": "gobsidian-installer", Accept: "application/vnd.github+json" };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
    log.info("Usando token do ambiente para acessar " + REPO + ".");
  }
  return headers;
}

async function getRelease(version) {
  const headers = githubHeaders();
  const uri = version
    ? `https://api.github.com/repos/${REPO}/releases/tags/${encodeURIComponent(version)}`
    : `https://api.github.com/repos/${REPO}/releases/latest`;
  if (!version) log.step(`Consultando a release mais recente de ${REPO}`);
  const res = await fetch(uri, { headers });
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(`GitHub respondeu ${res.status} ${res.statusText} para ${uri}\n${body.slice(0, 300)}`);
  }
  return { release: await res.json(), headers };
}

// Baixa um asset da release RESOLVENDO PELO ID na lista de assets, nunca
// montando a URL de /releases/download a mao (repo privado + redirect HTML
// silencioso em nome errado -- os mesmos dois motivos do install.ps1).
async function downloadReleaseAsset(release, headers, candidateNames, destino) {
  let asset = null;
  for (const n of candidateNames) {
    asset = release.assets.find((a) => a.name === n);
    if (asset) break;
  }
  if (!asset) {
    const disponiveis = release.assets.map((a) => a.name).join(", ") || "(nenhum asset)";
    throw new Error(
      `a release ${release.tag_name} nao traz nenhum de: ${candidateNames.join(", ")}. Ela publica: ${disponiveis}`
    );
  }
  const h = { ...headers, Accept: "application/octet-stream" };
  const res = await fetch(asset.url, { headers: h, redirect: "follow" });
  if (!res.ok) throw new Error(`download de ${asset.name} falhou: ${res.status} ${res.statusText}`);
  const buf = Buffer.from(await res.arrayBuffer());
  await fsp.writeFile(destino, buf);
  return asset.name;
}

// Nomes de asset esperados por plataforma/arquitetura. O repo nomeou a v0.1.0
// com hifen e a v1.0.0 com sublinhado -- aceitam-se as duas formas, como no
// instalador original. Para macOS/Linux a convencao e inferida do padrao
// windows_amd64; se a release publicar nomes diferentes, o erro acima lista
// os assets reais para ajuste.
function assetCandidates() {
  const archMap = { x64: "amd64", arm64: "arm64" };
  const arch = archMap[os.arch()] || os.arch();
  const osMap = { win32: "windows", darwin: "darwin", linux: "linux" };
  const plat = osMap[PLATFORM] || PLATFORM;
  const ext = IS_WIN ? ".exe" : "";
  return [
    `gobsidian_${plat}_${arch}${ext}`,
    `gobsidian-${plat}-${arch}${ext}`,
  ];
}

// -----------------------------------------------------------------------
// 2. Baixar e CONFERIR O HASH -- nao opcional
// -----------------------------------------------------------------------
async function sha256File(filePath) {
  const hash = crypto.createHash("sha256");
  await new Promise((resolve, reject) => {
    const stream = fs.createReadStream(filePath);
    stream.on("data", (d) => hash.update(d));
    stream.on("end", resolve);
    stream.on("error", reject);
  });
  return hash.digest("hex");
}

function findExpectedHash(sumsText, assetName) {
  for (const linha of sumsText.split("\n")) {
    const m = linha.match(/^\s*([0-9a-fA-F]{64})\s+\*?(?:.*[\\/])?(.+?)\s*$/);
    if (m && m[2] === assetName) return m[1].toLowerCase();
  }
  return null;
}

// -----------------------------------------------------------------------
// 3. Onde instalar
// -----------------------------------------------------------------------
function defaultInstallDir(admin) {
  if (IS_WIN) {
    if (admin) return path.join(process.env.ProgramFiles || "C:\\Program Files", "gobsidian");
    log.info("Sem elevacao: instalando por usuario. Rode como Administrador para instalar em Program Files.");
    return path.join(process.env.LOCALAPPDATA || path.join(os.homedir(), "AppData", "Local"), "Programs", "gobsidian");
  }
  if (admin) return "/usr/local/gobsidian";
  // XDG-ish: binario de usuario, sem exigir sudo.
  return path.join(os.homedir(), ".local", "share", "gobsidian");
}

// Processos rodando A PARTIR DESTE destino, com o cofre de cada um.
//
// Por caminho, nunca por nome: matar por nome ja derrubou a sessao de Claude de
// um usuario neste projeto, e um gobsidian instalado em outro lugar nao tem
// nada a ver com esta instalacao.
function listarEmUso(destino) {
  try {
    if (IS_WIN) {
      const ps = `Get-CimInstance Win32_Process -Filter "Name='gobsidian.exe'" | Where-Object { $_.ExecutablePath -eq '${destino.replace(/'/g, "''")}' } | ForEach-Object { "$($_.ProcessId)|$($_.CommandLine)" }`;
      const r = spawnSync("powershell", ["-NoProfile", "-Command", ps], { encoding: "utf8" });
      return separarLinhas(r.stdout).map((linha) => {
        const i = linha.indexOf("|");
        return { pid: linha.slice(0, i), cmd: linha.slice(i + 1) };
      });
    }
    const r = spawnSync("pgrep", ["-af", destino], { encoding: "utf8" });
    return separarLinhas(r.stdout).map((linha) => {
      const i = linha.indexOf(" ");
      return { pid: linha.slice(0, i), cmd: linha.slice(i + 1) };
    });
  } catch {
    return [];
  }
}

function separarLinhas(saida) {
  return String(saida || "").split(/\r?\n/).map((s) => s.trim()).filter(Boolean);
}

function cofreDe(cmd) {
  const m = cmd.match(/--vault\s+"([^"]+)"/) || cmd.match(/--vault\s+(\S+)/);
  return m ? m[1] : "(cofre nao identificado)";
}

function matar(pid) {
  if (IS_WIN) {
    const ps = `Stop-Process -Id ${pid} -Force -ErrorAction SilentlyContinue`;
    spawnSync("powershell", ["-NoProfile", "-Command", ps], { stdio: "ignore" });
  } else {
    spawnSync("kill", ["-9", String(pid)], { stdio: "ignore" });
  }
}

// Diretorio de runtime do daemon, o MESMO que internal/ipc.runtimeDir calcula.
function runtimeDir() {
  if (IS_WIN) return path.join(process.env.LOCALAPPDATA || "", "gobsidian", "run");
  if (process.env.XDG_RUNTIME_DIR) return path.join(process.env.XDG_RUNTIME_DIR, "gobsidian");
  const uid = typeof process.getuid === "function" ? process.getuid() : 0;
  return path.join(os.tmpdir(), `gobsidian-${uid}`);
}

function pidVivo(pid) {
  try {
    process.kill(pid, 0);
    return true;
  } catch (e) {
    // EPERM: o processo existe e e de outro usuario. ESRCH: nao existe.
    return e.code === "EPERM";
  }
}

// Remove lock de inicializacao do daemon cujo PID ja morreu.
//
// internal/daemon.adquirirLock cria o arquivo com O_CREATE|O_EXCL e grava o PID
// dentro, mas NUNCA le esse PID de volta: se o arquivo existe, a ponte conclui
// "outra ja esta subindo o daemon" e so espera. O `defer liberar()` que deveria
// remove-lo nao roda quando o processo e MORTO -- e matar processos e
// exatamente o que este instalador faz logo acima.
//
// Medido numa maquina real: um lock com PID morto deixou o daemon desligado por
// tres dias. Toda sessao MCP esperava 10 s em vao e depois construia o proprio
// indice, perdendo os -60% e -74% de memoria que o daemon existe para dar.
//
// Remove SO o que tem PID morto: lock com processo vivo e uma ponte legitima
// subindo o daemon agora, e apaga-lo abriria a corrida que o lock fecha.
function removerLocksObsoletos() {
  const dir = runtimeDir();
  let entradas;
  try {
    entradas = fs.readdirSync(dir).filter((f) => f.endsWith(".lock"));
  } catch {
    return;
  }
  let removidos = 0;
  for (const nome of entradas) {
    const alvo = path.join(dir, nome);
    let pid = 0;
    try {
      pid = parseInt(String(fs.readFileSync(alvo, "utf8")).trim(), 10);
    } catch {
      continue;
    }
    if (pid && pidVivo(pid)) {
      log.info(`Lock ${nome} pertence ao PID ${pid}, que esta vivo; mantido.`);
      continue;
    }
    try {
      fs.rmSync(alvo, { force: true });
      removidos++;
    } catch {
      /* melhor esforco */
    }
  }
  if (removidos > 0) log.ok(`${removidos} lock(s) obsoleto(s) do daemon removido(s)`);
}

// -----------------------------------------------------------------------
// 4. PATH
// -----------------------------------------------------------------------
function normPath(p) {
  return p.replace(/[\\/]+$/, "").toLowerCase();
}

function updatePathWindows(installDir, admin) {
  const hive = admin && installDir.toLowerCase().startsWith((process.env.ProgramFiles || "").toLowerCase())
    ? "HKLM\\SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment"
    : "HKCU\\Environment";
  let atual = "";
  try {
    const out = execFileSync("reg", ["query", hive, "/v", "Path"], { encoding: "utf8" });
    const m = out.match(/Path\s+REG_(?:EXPAND_)?SZ\s+(.*)/);
    atual = m ? m[1].trim() : "";
  } catch {
    atual = "";
  }
  const partes = atual.split(";").filter(Boolean);
  if (partes.some((p) => normPath(p) === normPath(installDir))) {
    log.info(`PATH (${hive.includes("HKLM") ? "Machine" : "User"}) ja contem ${installDir}`);
    return;
  }
  const novo = partes.concat(installDir).join(";");
  execFileSync("reg", ["add", hive, "/v", "Path", "/t", "REG_EXPAND_SZ", "/d", novo, "/f"], { stdio: "ignore" });
  log.ok(`PATH (${hive.includes("HKLM") ? "Machine" : "User"}) recebeu ${installDir}`);
  log.info("Terminais ja abertos so enxergam a mudanca depois de reabrir.");
  process.env.Path = `${process.env.Path || ""};${installDir}`;
}

function updatePathUnix(installDir) {
  const linha = `export PATH="${installDir}:$PATH"`;
  const candidatos = [".zshrc", ".bashrc", ".bash_profile", ".profile"]
    .map((f) => path.join(os.homedir(), f));
  const rc = candidatos.find((f) => fs.existsSync(f)) || candidatos[candidatos.length - 1];
  const atual = fs.existsSync(rc) ? fs.readFileSync(rc, "utf8") : "";
  if (atual.includes(installDir)) {
    log.info(`${rc} ja referencia ${installDir}`);
  } else {
    fs.appendFileSync(rc, `\n# gobsidian-installer\n${linha}\n`);
    log.ok(`PATH atualizado em ${rc}`);
    log.info("Abra um novo terminal (ou rode 'source " + rc + "') para carregar a mudanca.");
  }
  process.env.PATH = `${installDir}:${process.env.PATH || ""}`;
}

// -----------------------------------------------------------------------
// 5. Qual cofre
// -----------------------------------------------------------------------
function obsidianConfigPath() {
  if (IS_WIN) return path.join(process.env.APPDATA || "", "obsidian", "obsidian.json");
  if (PLATFORM === "darwin") return path.join(os.homedir(), "Library", "Application Support", "obsidian", "obsidian.json");
  return path.join(os.homedir(), ".config", "obsidian", "obsidian.json");
}

function getObsidianVaults() {
  const reg = obsidianConfigPath();
  if (!fs.existsSync(reg)) return [];
  try {
    const j = JSON.parse(fs.readFileSync(reg, "utf8"));
    const vaults = j.vaults || {};
    return Object.values(vaults)
      .map((v) => v.path)
      .filter((p) => p && fs.existsSync(p));
  } catch {
    return [];
  }
}

// -----------------------------------------------------------------------
// 6. Hosts de IA -- deteccao e configuracao
// -----------------------------------------------------------------------
function mergeMcpJson(filePath, serverKey, exe, args) {
  const dir = path.dirname(filePath);
  fs.mkdirSync(dir, { recursive: true });
  let doc = {};
  if (fs.existsSync(filePath)) {
    fs.copyFileSync(filePath, `${filePath}.gobsidian-backup`);
    const bruto = fs.readFileSync(filePath, "utf8").trim();
    if (bruto) doc = JSON.parse(bruto);
  }
  if (!doc.mcpServers) doc.mcpServers = {};
  doc.mcpServers[serverKey] = { command: exe, args };
  fs.writeFileSync(filePath, JSON.stringify(doc, null, 2), "utf8");
}

// Resolve o caminho real de um comando. No Windows `where` devolve varias
// linhas -- o shim sem extensao e o .cmd -- e a que interessa e a executavel.
function resolveCli(cmd) {
  const probe = IS_WIN ? "where" : "which";
  const r = spawnSync(probe, [cmd], { encoding: "utf8" });
  if (r.status !== 0 || !r.stdout) return null;
  const linhas = r.stdout.split(/\r?\n/).map((s) => s.trim()).filter(Boolean);
  if (!IS_WIN) return linhas[0] || null;
  const exec = linhas.find((l) => /\.(exe|cmd|bat)$/i.test(l));
  return exec || linhas[0] || null;
}

// Executa um CLI de host.
//
// No Windows, `claude`, `gemini` e `code` sao shims .cmd do npm, e
// spawnSync("claude", ...) sem shell NAO os executa: o processo nem nasce e o
// status volta `null`, que e como a primeira versao multi-cofre falhou --
// "claude mcp add saiu null", sem saida nenhuma para explicar.
//
// A saida nao e `shell: true`: com ele o Node junta os argumentos por espaco sem
// aspas, e o primeiro caminho de cofre com espaco quebraria tudo. O caminho
// certo e chamar cmd.exe passando a linha montada e citada por nos, com
// windowsVerbatimArguments para o Node nao citar de novo por cima.
function runCli(cmd, args) {
  const alvo = resolveCli(cmd);
  if (!alvo) {
    return { status: null, stdout: "", stderr: `comando nao encontrado no PATH: ${cmd}` };
  }
  if (IS_WIN && /\.(cmd|bat)$/i.test(alvo)) {
    const linha = [alvo, ...args].map(quoteWin).join(" ");
    return spawnSync("cmd.exe", ["/d", "/s", "/c", `"${linha}"`], {
      encoding: "utf8",
      windowsVerbatimArguments: true,
    });
  }
  return spawnSync(alvo, args, { encoding: "utf8" });
}

// Citação para linha de comando do Windows: aspas sempre, e as internas
// escapadas. Caminho de cofre com espaco e o caso comum, nao a excecao.
function quoteWin(s) {
  return `"${String(s).replace(/(\\*)"/g, '$1$1\\"').replace(/(\\*)$/, "$1$1")}"`;
}

function appDataPath(...parts) {
  // Equivalente cross-platform de %APPDATA% para apps Electron-like.
  if (IS_WIN) return path.join(process.env.APPDATA || "", ...parts);
  if (PLATFORM === "darwin") return path.join(os.homedir(), "Library", "Application Support", ...parts);
  return path.join(os.homedir(), ".config", ...parts);
}

function localAppDataPath(...parts) {
  if (IS_WIN) return path.join(process.env.LOCALAPPDATA || "", ...parts);
  if (PLATFORM === "darwin") return path.join(os.homedir(), "Library", "Application Support", ...parts);
  return path.join(os.homedir(), ".local", "share", ...parts);
}

// Os hosts NAO capturam mais o cofre: `configura` recebe a chave do servidor e
// os argumentos, e por isso o mesmo host pode ser configurado uma vez por cofre.
function buildHostCandidates() {
  return [
    {
      key: "claude-desktop",
      nome: "Claude Desktop",
      detecta: () => fs.existsSync(appDataPath("Claude")) || fs.existsSync(localAppDataPath("AnthropicClaude")),
      configura: (exe, serverKey, argsServe) => {
        mergeMcpJson(appDataPath("Claude", "claude_desktop_config.json"), serverKey, exe, argsServe);
        return "Reinicie o Claude Desktop para carregar o servidor.";
      },
    },
    {
      key: "claude-code",
      nome: "Claude Code (CLI)",
      detecta: () => testExe("claude"),
      configura: (exe, serverKey, argsServe) => {
        runCli("claude", ["mcp", "remove", serverKey, "--scope", "user"]);
        const r = runCli("claude", ["mcp", "add", serverKey, "--scope", "user", "--", exe, ...argsServe]);
        if (r.status !== 0) throw new Error(`claude mcp add saiu ${r.status}: ${(r.stdout || "") + (r.stderr || "")}`);
        return "Registrado no escopo de usuario.";
      },
    },
    {
      key: "gemini-cli",
      nome: "Gemini CLI",
      detecta: () => testExe("gemini"),
      configura: (exe, serverKey, argsServe) => {
        runCli("gemini", ["mcp", "remove", serverKey, "--scope", "user"]);
        const r = runCli("gemini", ["mcp", "add", serverKey, exe, ...argsServe, "--scope", "user"]);
        if (r.status !== 0) throw new Error(`gemini mcp add saiu ${r.status}: ${(r.stdout || "") + (r.stderr || "")}`);
        return "Registrado no escopo de usuario.";
      },
    },
    {
      key: "codex",
      nome: "Codex CLI",
      detecta: () => testExe("codex"),
      configura: (exe, serverKey, argsServe) => {
        runCli("codex", ["mcp", "remove", serverKey]);
        const r = runCli("codex", ["mcp", "add", serverKey, "--", exe, ...argsServe]);
        if (r.status !== 0) throw new Error(`codex mcp add saiu ${r.status}: ${(r.stdout || "") + (r.stderr || "")}`);
        return "Registrado em ~/.codex/config.toml.";
      },
    },
    {
      key: "vscode",
      nome: "VS Code",
      detecta: () => testExe("code"),
      configura: (exe, serverKey, argsServe) => {
        const def = JSON.stringify({ name: serverKey, command: exe, args: argsServe });
        const r = runCli("code", ["--add-mcp", def]);
        if (r.status !== 0) throw new Error(`code --add-mcp saiu ${r.status}: ${(r.stdout || "") + (r.stderr || "")}`);
        return "Registrado na configuracao de usuario do VS Code.";
      },
    },
    {
      key: "cursor",
      nome: "Cursor",
      detecta: () => testExe("cursor") || fs.existsSync(localAppDataPath("Programs", "cursor")) || fs.existsSync(path.join(os.homedir(), "Applications", "Cursor.app")),
      configura: (exe, serverKey, argsServe) => {
        mergeMcpJson(path.join(os.homedir(), ".cursor", "mcp.json"), serverKey, exe, argsServe);
        return "Reinicie o Cursor.";
      },
    },
    {
      key: "windsurf",
      nome: "Windsurf",
      detecta: () => fs.existsSync(path.join(os.homedir(), ".codeium", "windsurf")),
      configura: (exe, serverKey, argsServe) => {
        mergeMcpJson(path.join(os.homedir(), ".codeium", "windsurf", "mcp_config.json"), serverKey, exe, argsServe);
        return "Reinicie o Windsurf.";
      },
    },
    {
      key: "antigravity",
      nome: "Antigravity",
      detecta: () => fs.existsSync(localAppDataPath("Programs", "Antigravity")),
      configura: (exe, serverKey, argsServe) => {
        mergeMcpJson(path.join(os.homedir(), ".gemini", "antigravity", "mcp_config.json"), serverKey, exe, argsServe);
        return "Reinicie o Antigravity.";
      },
    },
    {
      key: "antigravity-ide",
      nome: "Antigravity IDE",
      detecta: () => fs.existsSync(localAppDataPath("Programs", "Antigravity IDE")),
      configura: (exe, serverKey, argsServe) => {
        mergeMcpJson(path.join(os.homedir(), ".gemini", "antigravity-ide", "mcp_config.json"), serverKey, exe, argsServe);
        return "Reinicie o Antigravity IDE.";
      },
    },
  ];
}

// -----------------------------------------------------------------------
// main
// -----------------------------------------------------------------------
async function main() {
  const opts = parseArgs(process.argv.slice(2));

  log.title("gobsidian - servidor MCP para cofres Obsidian");

  // 1) qual e a release mais recente ---------------------------------------
  let release, headers;
  try {
    ({ release, headers } = await getRelease(opts.version));
  } catch (e) {
    log.err(`Nao foi possivel consultar as releases de ${REPO}.`);
    log.info(`Detalhe: ${e.message}`);
    if (!process.env.GITHUB_TOKEN && !process.env.GH_TOKEN) {
      log.info("O GitHub devolve 404 tanto para release inexistente quanto para repositorio privado.");
      log.info("Se este for privado, defina um token e rode de novo: GITHUB_TOKEN=<token com escopo repo>");
    }
    process.exitCode = 1;
    return;
  }
  const version = release.tag_name;

  // 2) ja esta instalado? em que versao? -----------------------------------
  //
  // Reinstalar o que ja esta certo custa uma release inteira de download e nao
  // muda nada. O caso comum de rodar o instalador uma segunda vez e querer
  // acrescentar um cofre, nao trocar o binario.
  const admin = isAdmin();
  const installDir = opts.installDir || defaultInstallDir(admin);
  const jaInstalado = findInstalledExe(installDir);
  const atual = jaInstalado ? readInstalledVersion(jaInstalado) : { raw: null, tag: null };

  let destino = jaInstalado;
  let versao = atual.raw;
  let precisaInstalar = true;

  if (jaInstalado && atual.tag === version && !opts.force) {
    precisaInstalar = false;
    log.ok(`gobsidian ${version} ja instalado em ${jaInstalado}`);
    log.info("Versao mais recente. Pulando o download; indo direto para os cofres.");
    log.info("Use --force para reinstalar o binario mesmo assim.");
  } else if (jaInstalado && atual.tag && atual.tag !== version) {
    log.warn(`Instalado: ${atual.tag}. Disponivel: ${version}. Atualizando.`);
  } else if (jaInstalado && !atual.tag) {
    log.warn(`Ha um gobsidian em ${jaInstalado}, mas 'version' nao respondeu. Reinstalando.`);
  } else {
    log.ok(`Versao a instalar: ${version}`);
  }

  // 3) baixar, conferir hash e instalar ------------------------------------
  if (precisaInstalar) {
    const tmpDir = await fsp.mkdtemp(path.join(os.tmpdir(), "gobsidian_install_"));
    try {
      const tmpExe = path.join(tmpDir, EXE_NAME);
      log.step(`Baixando o binario (${PLATFORM}/${os.arch()})`);
      let assetName;
      try {
        assetName = await downloadReleaseAsset(release, headers, assetCandidates(), tmpExe);
      } catch (e) {
        log.err("Nao foi possivel baixar o binario.");
        log.info(e.message);
        process.exitCode = 1;
        return;
      }
      log.ok(`Baixado: ${assetName}`);

      log.step("Conferindo SHA-256 contra SHA256SUMS.txt");
      const tmpSums = path.join(tmpDir, "SHA256SUMS.txt");
      let sums;
      try {
        await downloadReleaseAsset(release, headers, ["SHA256SUMS.txt"], tmpSums);
        sums = await fsp.readFile(tmpSums, "utf8");
      } catch {
        log.err("SHA256SUMS.txt nao foi encontrado nesta release. Instalacao abortada.");
        log.info("Um binario sem soma publicada nao pode ser verificado, e instalar assim seria pior do que nao instalar.");
        process.exitCode = 1;
        return;
      }
      const esperado = findExpectedHash(sums, assetName);
      if (!esperado) {
        log.err(`SHA256SUMS.txt nao traz uma linha para ${assetName}. Instalacao abortada.`);
        process.exitCode = 1;
        return;
      }
      const obtido = await sha256File(tmpExe);
      if (obtido !== esperado) {
        log.err("SHA-256 NAO CONFERE. Instalacao abortada.");
        log.info(`  esperado: ${esperado}`);
        log.info(`  obtido:   ${obtido}`);
        process.exitCode = 1;
        return;
      }
      log.ok(`SHA-256 confere (${esperado.slice(0, 16)}...)`);

      destino = path.join(installDir, EXE_NAME);

      // PERGUNTA antes de matar. Quem esta rodando gobsidian esta com sessoes
      // MCP abertas; encerrar sem avisar derruba trabalho em curso e o usuario
      // nao tem como saber que foi o instalador. --yes ou ambiente nao
      // interativo seguem sem perguntar, que e a semantica do resto do script.
      const emUso = listarEmUso(destino);
      if (emUso.length > 0) {
        log.warn(`${emUso.length} processo(s) gobsidian rodando a partir de ${destino}:`);
        for (const p of emUso) log.info(`    PID ${p.pid}  ${cofreDe(p.cmd)}`);
        log.info("  O binario nao pode ser substituido enquanto eles estiverem abertos.");
        let encerrar = true;
        if (isInteractive(opts)) {
          const r = await ask("  Encerrar esses processos e continuar? [s/N] ");
          encerrar = /^[sy]/i.test(r);
        }
        if (!encerrar) {
          log.err("Instalacao cancelada: os processos acima continuam rodando.");
          log.info("Feche o host MCP (Claude Desktop, Claude Code) e rode o instalador de novo.");
          process.exitCode = 1;
          return;
        }
        for (const p of emUso) matar(p.pid);
      }

      removerLocksObsoletos();

      log.step(`Instalando em ${installDir}`);
      fs.mkdirSync(installDir, { recursive: true });
      fs.copyFileSync(tmpExe, destino);
      if (!IS_WIN) fs.chmodSync(destino, 0o755);
      log.ok(`Binario em ${destino}`);
      try {
        versao = execFileSync(destino, ["version"], { encoding: "utf8" }).trim();
        log.info(versao);
      } catch {
        versao = "(nao foi possivel executar version)";
        log.warn(versao);
      }

      if (!opts.noPath) {
        try {
          if (IS_WIN) updatePathWindows(installDir, admin);
          else updatePathUnix(installDir);
        } catch (e) {
          log.warn(`Nao foi possivel atualizar o PATH automaticamente: ${e.message}`);
          log.info(`Adicione manualmente ao PATH: ${installDir}`);
        }
      }
    } finally {
      await fsp.rm(tmpDir, { recursive: true, force: true });
    }
  }

  // 4) quais cofres ---------------------------------------------------------
  let vaults = opts.vaults.slice();
  const encontrados = getObsidianVaults();
  if (!vaults.length) {
    if (isInteractive(opts)) {
      vaults = await promptVaults(encontrados);
    } else if (encontrados.length === 1) {
      vaults = [encontrados[0]];
    }
  }

  // Valida e normaliza. Um cofre invalido nao derruba os outros: quem passou
  // tres caminhos e errou um quer os dois que estao certos.
  const validos = [];
  for (const v of vaults) {
    if (!fs.existsSync(v) || !fs.statSync(v).isDirectory()) {
      log.err(`Cofre nao encontrado, ignorando: ${v}`);
      continue;
    }
    const real = fs.realpathSync(v);
    if (!fs.existsSync(path.join(real, ".obsidian"))) {
      log.warn(`Nao ha .obsidian em ${real} - pode nao ser um cofre do Obsidian.`);
    }
    validos.push(real);
  }
  const cofres = [...new Set(validos)];

  if (!cofres.length) {
    log.err("Nenhum cofre indicado. Passe --vault (repetivel) ou defina GOBSIDIAN_VAULT.");
    process.exitCode = 1;
    return;
  }
  if (opts.name && cofres.length > 1) {
    log.warn("--name vale para um cofre so; com varios, o nome sai do nome da pasta de cada um.");
  }

  // Um servidor MCP por cofre, com nome proprio.
  const servidores = cofres.map((v) => {
    const args = ["serve", "--vault", v];
    if (opts.readOnly) args.push("--read-only");
    return {
      vault: v,
      key: serverKeyForVault(v, cofres.length, cofres.length === 1 ? opts.name : null),
      args,
    };
  });

  console.log("");
  for (const s of servidores) log.ok(`Cofre: ${s.vault}  ->  servidor "${s.key}"`);

  // 5) hosts de IA ----------------------------------------------------------
  log.step("Procurando hosts de IA");
  const candidatos = buildHostCandidates();
  const detectados = candidatos.filter((h) => {
    try { return h.detecta(); } catch { return false; }
  });

  if (!detectados.length) {
    log.warn("Nenhum host conhecido foi detectado.");
    log.info("O binario esta instalado. Configure seu host manualmente com:");
    for (const s of servidores) {
      log.info(`  ${s.key}: comando ${destino}, args ${s.args.join(" ")}`);
    }
    return;
  }
  for (const h of detectados) log.ok(`encontrado: ${h.nome}`);

  let escolhidos = [];
  if (opts.hosts) {
    escolhidos = detectados.filter((h) => opts.hosts.includes(h.key));
    for (const k of opts.hosts) {
      if (!detectados.some((h) => h.key === k)) log.warn(`host pedido mas nao detectado: ${k}`);
    }
  } else if (!isInteractive(opts)) {
    log.warn("Modo nao interativo sem --hosts: nenhum host sera configurado.");
    log.info(`Passe --hosts com as chaves: ${detectados.map((h) => h.key).join(", ")}`);
  } else {
    escolhidos = await promptHostChoice(detectados);
  }

  // 6) configurar: cada host, uma vez por cofre ------------------------------
  const ok = [];
  const falhou = [];
  for (const h of escolhidos) {
    for (const s of servidores) {
      log.step(`Configurando ${h.nome} -> ${s.key}`);
      try {
        const msg = h.configura(destino, s.key, s.args);
        log.ok(`${h.nome} / ${s.key} - ${msg}`);
        ok.push(`${h.nome}:${s.key}`);
      } catch (e) {
        // Um par host/cofre que falha nao pode derrubar os demais.
        log.err(`${h.nome} / ${s.key}: ${e.message}`);
        falhou.push(`${h.nome}:${s.key}`);
      }
    }
  }

  // 7) resumo ---------------------------------------------------------------
  log.title("Resumo");
  console.log(`   binario    ${destino}`);
  console.log(`   versao     ${versao}`);
  if (!precisaInstalar) console.log("   download   pulado; a instalacao ja estava na versao mais recente");
  for (const s of servidores) console.log(`   cofre      ${s.key}  ->  ${s.vault}`);
  if (opts.readOnly) console.log("   modo       somente leitura (--read-only)");
  if (ok.length) console.log(paint(ANSI.green, `   ok         ${ok.join(", ")}`));
  if (falhou.length) console.log(paint(ANSI.red, `   falhou     ${falhou.join(", ")}`));
  console.log("");
  if (ok.length) log.info("Reinicie os hosts configurados para que carreguem os servidores.");
  log.info("Se algo nao funcionar, o primeiro comando e:");
  console.log(`   gobsidian doctor --vault "${servidores[0].vault}"`);
  console.log("");
}


main().catch((e) => {
  log.err(e.stack || e.message);
  process.exitCode = 1;
});
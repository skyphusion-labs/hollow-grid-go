package transport

import (
	"html"
	"strings"
)

// playPage returns the browser xterm play client HTML (TS parity: the-hollow-grid webclient.ts).
func playPage(worldName string) string {
	title := worldName
	if title == "" {
		title = "The Hollow Grid"
	}
	title = html.EscapeString(title)
	return strings.ReplaceAll(playPageTemplate, "__TITLE__", title)
}

const playPageTemplate = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>__TITLE__</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/xterm@5.3.0/css/xterm.min.css" />
<style>
  html, body { height: 100%; margin: 0; background: #08080b; color: #c9c9cf;
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }
  #wrap { display: flex; flex-direction: column; height: 100%; }
  header { display: flex; justify-content: space-between; align-items: center;
    padding: 8px 14px; border-bottom: 1px solid #18181f; font-size: 13px; color: #6aa3a3; }
  header .net { color: #4a4a55; }
  header a.policies { color: #6aa3a3; text-decoration: none; }
  header a.policies:hover { text-decoration: underline; }
  #status { font-size: 12px; color: #888; }
  #term { flex: 1; min-height: 0; padding: 6px 4px 4px 10px; }
  .term-wrap { height: 100%; }
</style>
</head>
<body>
<div id="wrap">
  <header>
    <span>__TITLE__ <span class="net">:: the hollow grid network</span> <a class="net policies" href="https://github.com/skyphusion-labs/the-hollow-grid/tree/main/docs/legal" target="_blank" rel="noopener">policies</a></span>
    <span id="status">connecting&hellip;</span>
  </header>
  <div id="term" class="term-wrap"></div>
</div>
<script src="https://cdn.jsdelivr.net/npm/xterm@5.3.0/lib/xterm.min.js"></script>
<script src="https://cdn.jsdelivr.net/npm/xterm-addon-fit@0.8.0/lib/xterm-addon-fit.min.js"></script>
<script>
  const statusEl = document.getElementById("status");
  const setStatus = (text, color) => { statusEl.textContent = text; statusEl.style.color = color; };

  const term = new Terminal({
    cursorBlink: true,
    fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
    fontSize: 14,
    scrollback: 4000,
    theme: { background: "#08080b", foreground: "#c9c9cf", cursor: "#6aa3a3" },
  });
  const fit = new FitAddon.FitAddon();
  term.loadAddon(fit);
  term.open(document.getElementById("term"));
  fit.fit();
  addEventListener("resize", () => fit.fit());

  const proto = location.protocol === "https:" ? "wss" : "ws";
  const ws = new WebSocket(proto + "://" + location.host + "/ws");

  let buf = "";
  ws.onmessage = (e) => {
    if (typeof e.data !== "string") return;
    buf += e.data;
    const lines = buf.split(/\r?\n/);
    buf = lines.pop();
    for (const line of lines) {
      if (line.startsWith("@event ")) continue;
      term.write(line + "\r\n");
    }
  };
  ws.onopen = () => { setStatus("connected", "#6c6"); term.focus(); };
  ws.onerror = () => setStatus("connection error", "#c66");
  ws.onclose = () => {
    setStatus("disconnected", "#c66");
    term.write("\r\n\x1b[90m[connection closed -- refresh to reconnect]\x1b[0m\r\n");
  };

  let lineBuf = "";
  term.onData((d) => {
    if (ws.readyState !== WebSocket.OPEN) return;
    for (const ch of d) {
      if (ch === "\r") { ws.send(lineBuf); term.write("\r\n"); lineBuf = ""; }
      else if (ch === "\u007f") { if (lineBuf.length) { lineBuf = lineBuf.slice(0, -1); term.write("\b \b"); } }
      else if (ch === "\u0003") { term.write("^C\r\n"); lineBuf = ""; }
      else if (ch >= " ") { lineBuf += ch; term.write(ch); }
    }
  });
</script>
</body>
</html>`

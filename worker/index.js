import { DurableObject } from "cloudflare:workers";

const PORT = 8790;
const IDLE_MS = 30 * 60 * 1000;
const SNAPSHOT_MS = 5 * 60 * 1000;

// One Durable Object owns the Rust Choir container.
// The object starts the existing image, forwards the socket, and keeps a filesystem snapshot.
export class RustChoirWorld extends DurableObject {
  constructor(ctx, env) {
    super(ctx, env);
    this.starting = undefined;
    const container = ctx.container;
    if (container.running) {
      void ctx.blockConcurrencyWhile(() => container.setInactivityTimeout(IDLE_MS));
      void this.armSnapshot();
    }
  }

  async fetch(request) {
    const envVars = this.containerEnv();
    if (envVars instanceof Response) {
      return envVars;
    }
    this.starting ??= this.startAndWait(envVars).finally(() => {
      this.starting = undefined;
    });
    await this.starting;
    const url = new URL(request.url);
    url.protocol = "http:";
    url.host = "container";
    const forwarded = new Request(url, request);
    forwarded.headers.delete("host");
    return this.ctx.container.getTcpPort(PORT).fetch(forwarded);
  }

  async alarm() {
    const container = this.ctx.container;
    if (!container.running) {
      return;
    }
    try {
      const snapshot = await container.snapshotContainer({});
      await this.ctx.storage.put("disk", {
        snapshot,
        image: container.images.base,
      });
    } catch (error) {
      console.log("snapshot failed", error);
    }
    await this.ctx.storage.setAlarm(Date.now() + SNAPSHOT_MS);
  }

  containerEnv() {
    if (!this.env.ADMIN_TOKEN) {
      return new Response("ADMIN_TOKEN is not set.\n", { status: 503 });
    }
    if (!this.env.GRID_HUB_TOKEN) {
      return new Response("GRID_HUB_TOKEN is not set.\n", { status: 503 });
    }
    if (!this.env.GRID_WORLD_KEY) {
      return new Response("GRID_WORLD_KEY is not set.\n", { status: 503 });
    }
    return {
      LISTEN_ADDR: "0.0.0.0:8790",
      DATA_DIR: "/data",
      WORLD_NAME: "Rust Choir",
      WORLD_URL: "wss://rustchoir.skyphusion.org/ws",
      GRID_HUB_URL: "https://grid-hub.skyphusion.org/rpc",
      GRID_HUB_TOKEN: this.env.GRID_HUB_TOKEN,
      GRID_WORLD_KEY: this.env.GRID_WORLD_KEY,
      ADMINS: "skyphusion",
      ADMIN_TOKEN: this.env.ADMIN_TOKEN,
    };
  }

  async armSnapshot() {
    const current = await this.ctx.storage.getAlarm();
    if (current === null) {
      await this.ctx.storage.setAlarm(Date.now() + SNAPSHOT_MS);
    }
  }

  async startAndWait(envVars) {
    const container = this.ctx.container;
    if (!container.running) {
      const saved = await this.ctx.storage.get("disk");
      const image = container.images.base;
      if (saved && saved.image === image) {
        container.start({
          containerSnapshot: saved.snapshot,
          enableInternet: true,
          env: envVars,
        });
      } else {
        container.start({
          image,
          instance: "lite",
          enableInternet: true,
          env: envVars,
        });
      }
    }
    await container.setInactivityTimeout(IDLE_MS);
    await this.armSnapshot();
    const port = container.getTcpPort(PORT);
    let lastError;
    for (let attempt = 0; attempt < 150; attempt++) {
      try {
        const response = await port.fetch("http://container/health", {
          signal: AbortSignal.timeout(1000),
        });
        await response.body?.cancel();
        if (!response.ok) {
          throw new Error("Health check returned " + response.status);
        }
        return;
      } catch (error) {
        lastError = error;
        await scheduler.wait(200);
      }
    }
    throw new Error("Container did not become ready on port " + PORT, { cause: lastError });
  }
}

export default {
  fetch(request, env) {
    return env.WORLD.getByName("world").fetch(request);
  },
};

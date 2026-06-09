import { afterEach, describe, expect, test } from "bun:test";
import { existsSync } from "node:fs";
import { chromium, type Browser } from "playwright";

let browser: Browser | undefined;

describe("service worker browser isolation", () => {
  afterEach(async () => {
    await browser?.close();
    browser = undefined;
  });

  test("honors exact scope and rejects broader Service-Worker-Allowed paths", async () => {
    const server = Bun.serve({
      port: 0,
      fetch(req) {
        const path = new URL(req.url).pathname;
        if (path === "/") return html();
        if (path === "/__tinkabot_session/session-001/sw.js") {
          return js("session-001", {
            "Service-Worker-Allowed": "/__tinkabot_session/session-001/",
          });
        }
        if (path === "/__tinkabot_session/session-001/broad-sw.js") {
          return new Response("denied", { status: 403 });
        }
        return new Response("not found", { status: 404 });
      },
    });
    try {
      browser = await chromium.launch({
        executablePath: chrome(),
        headless: true,
        args: ["--no-sandbox"],
      });
      const page = await browser.newPage();
      await page.goto(`http://127.0.0.1:${server.port}/`);

      expect(await page.evaluate(() => "serviceWorker" in navigator)).toBe(true);

      const scope = await page.evaluate(async () => {
        const reg = await navigator.serviceWorker.register(
          "/__tinkabot_session/session-001/sw.js",
          { scope: "/__tinkabot_session/session-001/" },
        );
        await reg.unregister();
        return reg.scope;
      });
      expect(scope).toEndWith("/__tinkabot_session/session-001/");

      const wrongScope = await page.evaluate(async () => {
        try {
          await navigator.serviceWorker.register("/__tinkabot_session/session-001/sw.js", {
            scope: "/",
          });
          return "accepted";
        } catch (err) {
          return err instanceof Error ? err.name : String(err);
        }
      });
      expect(wrongScope).not.toBe("accepted");

      const broad = await page.evaluate(async () => {
        try {
          await navigator.serviceWorker.register(
            "/__tinkabot_session/session-001/broad-sw.js",
            { scope: "/" },
          );
          return "accepted";
        } catch (err) {
          return err instanceof Error ? err.name : String(err);
        }
      });
      expect(broad).not.toBe("accepted");
    } finally {
      server.stop(true);
    }
  });
});

function html() {
  return new Response("<!doctype html><title>sw proof</title>", {
    headers: { "Content-Type": "text/html" },
  });
}

function js(session: string, headers: Record<string, string>) {
  return new Response(`self.__tb_session = ${JSON.stringify(session)};`, {
    headers: {
      "Content-Type": "text/javascript",
      "Cache-Control": "no-store",
      "X-Tinkabot-Worker-Rev": "worker.rev.1",
      ...headers,
    },
  });
}

function chrome() {
  const path = process.env.PLAYWRIGHT_CHROME ?? "/usr/bin/google-chrome";
  if (!existsSync(path)) throw new Error(`missing browser: ${path}`);
  return path;
}

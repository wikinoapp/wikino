import { expect, test } from "@playwright/test";
import { createTestPage, createTestTopic, loadSharedTestData, query } from "../helpers/database";

test.use({ storageState: { cookies: [], origins: [] } });

const readMethods = ["GET", "HEAD"];

test("Cookieを使わないルートが各メソッドに固有の応答を返すこと", async ({ request }) => {
  const { space } = loadSharedTestData();
  const topic = await createTestTopic(space.id);
  const topicRecord = await query("SELECT number FROM topics WHERE id = $1 AND space_id = $2", [topic.id, space.id]);
  const page = await createTestPage(space.id, topic.id, { title: "配信テスト" });
  const spacePath = `/s/${space.identifier}`;
  const imagePath = `${spacePath}/pages/${page.number}/og_image/`;

  const routes = [
    { path: `${imagePath}old.png`, methods: readMethods, status: 302 },
    { path: `${spacePath}/atom`, methods: readMethods, status: 200 },
    { path: `${spacePath}/topics/${topicRecord.rows[0].number}/atom`, methods: readMethods, status: 200 },
    { path: "/api/v1/openapi.yaml", methods: ["GET"], status: 200 },
    { path: "/api/reference/v1", methods: readMethods, status: 200 },
    { path: "/oauth/token", methods: ["POST"], status: 400 },
    { path: "/oauth/revoke", methods: ["POST"], status: 400 },
    { path: "/oauth/device_authorization", methods: ["POST"], status: 400 },
    { path: "/.well-known/oauth-authorization-server", methods: readMethods, status: 200 },
    {
      path: `/.well-known/oauth-protected-resource/api/v1/spaces/${space.identifier}`,
      methods: readMethods,
      status: 200,
    },
    { path: "/.well-known/oauth-protected-resource/mcp", methods: readMethods, status: 200 },
    { path: "/.well-known/api-catalog", methods: readMethods, status: 200 },
    { path: "/mcp", methods: ["POST"], status: 401 },
    { path: "/mcp", methods: ["GET"], status: 405 },
  ];

  for (const { path, methods, status } of routes) {
    for (const method of methods) {
      await test.step(`${method} ${path}`, async () => {
        const response = await request.fetch(path, { method, maxRedirects: 0 });
        expect(response.status()).toBe(status);
        expect(response.headers()["set-cookie"]).toBeUndefined();
        if (status === 302) {
          expect(response.headers()["location"]).toContain(imagePath);
        }
      });
    }
  }
});

for (const method of readMethods) {
  test(`${method}で存在しない添付画像を要求してもCookieを発行しないこと`, async ({ request }) => {
    const response = await request.fetch("/attachments/550e8400-e29b-41d4-a716-446655440000/og_image", { method });

    // imgproxyの設定に応じて404または500になる。キャッシュ禁止のヘッダーで、画像のハンドラーの応答を確かめる。
    expect([404, 500]).toContain(response.status());
    expect(response.headers()["cache-control"]).toBe("private, no-store");
    expect(response.headers()["set-cookie"]).toBeUndefined();
  });
}

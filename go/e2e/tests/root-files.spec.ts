import { expect, test } from "@playwright/test";

test.use({ storageState: { cookies: [], origins: [] } });

const rootFiles = [
  { path: "/favicon.ico", mediaType: "image/vnd.microsoft.icon" },
  { path: "/apple-touch-icon.png", mediaType: "image/png" },
  { path: "/apple-touch-icon-precomposed.png", mediaType: "image/png" },
  { path: "/robots.txt", mediaType: "text/plain" },
];

for (const { path, mediaType } of rootFiles) {
  for (const method of ["GET", "HEAD"]) {
    test(`${method} ${path}は認証なしで配信され、Cookieを発行しないこと`, async ({ request }) => {
      const response = await request.fetch(path, { method, maxRedirects: 0 });

      expect(response.status()).toBe(200);
      expect(response.headers()["content-type"]?.split(";")[0]).toBe(mediaType);
      expect(response.headers()["set-cookie"]).toBeUndefined();
    });
  }
}

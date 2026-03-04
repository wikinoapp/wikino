import pg from "pg";
import * as fs from "fs";
import * as path from "path";

function getDatabaseURL(): string {
  const url = process.env.DATABASE_URL;
  if (!url) {
    throw new Error("環境変数 DATABASE_URL が設定されていません");
  }
  return url;
}

let pool: pg.Pool | null = null;

function getPool(): pg.Pool {
  if (!pool) {
    pool = new pg.Pool({
      connectionString: getDatabaseURL(),
      allowExitOnIdle: true,
    });
  }
  return pool;
}

export async function closePool(): Promise<void> {
  if (pool) {
    await pool.end();
    pool = null;
  }
}

export async function query(text: string, params?: unknown[]): Promise<pg.QueryResult> {
  return getPool().query(text, params);
}

async function queryWithRetry(text: string, params?: unknown[], maxRetries: number = 5): Promise<pg.QueryResult> {
  for (let attempt = 0; attempt < maxRetries; attempt++) {
    try {
      return await getPool().query(text, params);
    } catch (err: unknown) {
      const isUniqueViolation = err instanceof Error && "code" in err && (err as { code: string }).code === "23505";
      if (!isUniqueViolation || attempt === maxRetries - 1) {
        throw err;
      }
    }
  }
  throw new Error("queryWithRetry: unreachable");
}

export interface TestUser {
  id: string;
  email: string;
  atname: string;
  password: string;
}

export interface TestSpace {
  id: string;
  identifier: string;
  name: string;
}

export interface TestTopic {
  id: string;
  name: string;
}

export interface TestPage {
  id: string;
  number: number;
}

/**
 * テスト用ユーザーを作成する
 * bcryptハッシュは事前計算済みの "passw0rd" 用ハッシュを使用
 */
export async function createTestUser(
  overrides: Partial<{ email: string; atname: string; password: string }> = {},
): Promise<TestUser> {
  const suffix = Date.now().toString(36) + Math.random().toString(36).slice(2, 8);
  const email = overrides.email || `e2e-${suffix}@example.com`;
  const atname = overrides.atname || `e2e_${suffix.slice(0, 16)}`;
  const password = overrides.password || "passw0rd";

  // bcrypt hash of "passw0rd" (cost=4 for speed)
  const passwordDigest = "$2a$04$LjuVYgRWxYQS4BdhyX95wuwlnFxgJfm6sj.t2tXHTkYDE0Ir1rPrC";

  const userResult = await query(
    `INSERT INTO users (email, atname, name, description, locale, time_zone, joined_at, created_at, updated_at)
     VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW(), NOW())
     RETURNING id`,
    [email, atname, "E2E Test User", "", 0, "Asia/Tokyo"],
  );
  const id = userResult.rows[0].id;

  await query(
    `INSERT INTO user_passwords (user_id, password_digest, created_at, updated_at)
     VALUES ($1, $2, NOW(), NOW())`,
    [id, passwordDigest],
  );

  return { id, email, atname, password };
}

export async function createTestSpace(
  overrides: Partial<{ identifier: string; name: string }> = {},
): Promise<TestSpace> {
  const suffix = Date.now().toString(36) + Math.random().toString(36).slice(2, 8);
  const identifier = overrides.identifier || `e2e-sp-${suffix.slice(0, 12)}`;
  const name = overrides.name || `E2Eテストスペース ${suffix.slice(0, 8)}`;

  const result = await query(
    `INSERT INTO spaces (identifier, name, plan, joined_at, created_at, updated_at)
     VALUES ($1, $2, $3, NOW(), NOW(), NOW())
     RETURNING id`,
    [identifier, name, 1],
  );
  const id = result.rows[0].id;

  return { id, identifier, name };
}

export async function createTestSpaceMember(spaceId: string, userId: string, role: number = 0): Promise<string> {
  const result = await query(
    `INSERT INTO space_members (space_id, user_id, role, joined_at, active, created_at, updated_at)
     VALUES ($1, $2, $3, NOW(), true, NOW(), NOW())
     RETURNING id`,
    [spaceId, userId, role],
  );

  return result.rows[0].id;
}

export async function createTestTopic(spaceId: string, overrides: Partial<{ name: string }> = {}): Promise<TestTopic> {
  const suffix = Date.now().toString(36) + Math.random().toString(36).slice(2, 8);
  const name = overrides.name || `E2Eテストトピック ${suffix.slice(0, 8)}`;

  const result = await queryWithRetry(
    `INSERT INTO topics (space_id, number, name, description, visibility, created_at, updated_at)
     VALUES ($1, (SELECT COALESCE(MAX(number), 0) + 1 FROM topics WHERE space_id = $1), $2, $3, $4, NOW(), NOW())
     RETURNING id`,
    [spaceId, name, "", 0],
  );

  return { id: result.rows[0].id, name };
}

export async function createTestTopicMember(
  spaceId: string,
  topicId: string,
  spaceMemberId: string,
  role: number = 0,
): Promise<string> {
  const result = await query(
    `INSERT INTO topic_members (space_id, topic_id, space_member_id, role, joined_at, created_at, updated_at)
     VALUES ($1, $2, $3, $4, NOW(), NOW(), NOW())
     RETURNING id`,
    [spaceId, topicId, spaceMemberId, role],
  );

  return result.rows[0].id;
}

export async function createTestPage(
  spaceId: string,
  topicId: string,
  overrides: Partial<{ title: string; body: string }> = {},
): Promise<TestPage> {
  const title = overrides.title || null;
  const body = overrides.body || "";
  const bodyHtml = body ? `<p>${body}</p>` : "";

  const result = await queryWithRetry(
    `INSERT INTO pages (space_id, topic_id, number, title, body, body_html, linked_page_ids, modified_at, published_at, created_at, updated_at)
     VALUES ($1, $2, (SELECT COALESCE(MAX(number), 0) + 1 FROM pages WHERE space_id = $1), $3, $4, $5, $6, NOW(), NOW(), NOW(), NOW())
     RETURNING id, number`,
    [spaceId, topicId, title, body, bodyHtml, "{}"],
  );

  return { id: result.rows[0].id, number: result.rows[0].number };
}

export interface SharedTestData {
  user: TestUser;
  space: TestSpace;
  spaceMemberId: string;
}

const sharedTestDataPath = path.join(__dirname, "../playwright/.auth/test-data.json");

export function saveSharedTestData(data: SharedTestData): void {
  const dir = path.dirname(sharedTestDataPath);
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }
  fs.writeFileSync(sharedTestDataPath, JSON.stringify(data));
}

export function loadSharedTestData(): SharedTestData {
  return JSON.parse(fs.readFileSync(sharedTestDataPath, "utf-8"));
}

export async function createTestFeatureFlag(userId: string, name: string): Promise<void> {
  await query(
    `INSERT INTO feature_flags (user_id, name, created_at)
     VALUES ($1, $2, NOW())
     ON CONFLICT (user_id, name) DO NOTHING`,
    [userId, name],
  );
}

/**
 * テストデータを一括削除する
 * E2Eテストで作成したレコードをテスト後にクリーンアップする
 */
export async function cleanupTestData(userIds: string[]): Promise<void> {
  if (userIds.length === 0) return;

  // スペースメンバー経由でスペースIDを取得
  const spaceMemberResult = await query(`SELECT DISTINCT space_id FROM space_members WHERE user_id = ANY($1)`, [
    userIds,
  ]);
  const spaceIds = spaceMemberResult.rows.map((row: { space_id: string }) => row.space_id);

  if (spaceIds.length > 0) {
    // スペースに紐づくデータを削除（依存関係の順序に注意）
    await query(`DELETE FROM draft_pages WHERE space_id = ANY($1)`, [spaceIds]);
    await query(
      `DELETE FROM page_attachment_references WHERE page_id IN (SELECT id FROM pages WHERE space_id = ANY($1))`,
      [spaceIds],
    );
    await query(`DELETE FROM pages WHERE space_id = ANY($1)`, [spaceIds]);
    await query(`DELETE FROM topic_members WHERE space_id = ANY($1)`, [spaceIds]);
    await query(`DELETE FROM topics WHERE space_id = ANY($1)`, [spaceIds]);
    await query(`DELETE FROM space_members WHERE space_id = ANY($1)`, [spaceIds]);
    await query(`DELETE FROM spaces WHERE id = ANY($1)`, [spaceIds]);
  }

  // ユーザー関連データを削除
  await query(`DELETE FROM feature_flags WHERE user_id = ANY($1)`, [userIds]);
  await query(`DELETE FROM user_sessions WHERE user_id = ANY($1)`, [userIds]);
  await query(`DELETE FROM user_passwords WHERE user_id = ANY($1)`, [userIds]);
  await query(`DELETE FROM users WHERE id = ANY($1)`, [userIds]);
}

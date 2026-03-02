import { closePool, cleanupTestData, loadSharedTestData } from "./helpers/database";
import * as fs from "fs";
import * as path from "path";

async function globalTeardown() {
  const testDataFile = path.join(__dirname, "playwright/.auth/test-data.json");
  if (fs.existsSync(testDataFile)) {
    const data = loadSharedTestData();
    if (data.user?.id) {
      await cleanupTestData([data.user.id]);
    }
  }

  await closePool();
}

export default globalTeardown;

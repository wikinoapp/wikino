import { FileChecksum } from "@rails/activestorage/src/file_checksum";

/**
 * ファイルのMD5チェックサムをBase64形式で計算
 * @param file 計算対象のファイル
 * @returns Base64エンコードされたMD5チェックサム
 */
export async function calculateFileChecksum(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    FileChecksum.create(file, (error: Error | null, checksum?: string) => {
      if (error) {
        reject(error);
      } else if (checksum) {
        resolve(checksum);
      } else {
        reject(new Error("チェックサムの計算に失敗しました"));
      }
    });
  });
}

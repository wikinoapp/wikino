import SparkMD5 from "spark-md5";

/**
 * ファイルのMD5チェックサムをBase64形式で計算
 * @param file 計算対象のファイル
 * @returns Base64エンコードされたMD5チェックサム
 */
export async function calculateFileChecksum(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const chunkSize = 2097152; // 2MB
    const spark = new SparkMD5.ArrayBuffer();
    const fileReader = new FileReader();
    let currentChunk = 0;
    const chunks = Math.ceil(file.size / chunkSize);

    fileReader.onload = (e) => {
      if (e.target?.result) {
        spark.append(e.target.result as ArrayBuffer);
      }
      currentChunk++;

      if (currentChunk < chunks) {
        loadNext();
      } else {
        // MD5ハッシュを計算してBase64エンコード
        const rawHash = spark.end(true);
        // 16進数文字列をバイト配列に変換
        const bytes = [];
        for (let i = 0; i < rawHash.length; i += 2) {
          bytes.push(parseInt(rawHash.substr(i, 2), 16));
        }
        // Base64エンコード
        const base64 = btoa(String.fromCharCode(...bytes));
        resolve(base64);
      }
    };

    fileReader.onerror = () => {
      reject(new Error("ファイルの読み込みに失敗しました"));
    };

    function loadNext() {
      const start = currentChunk * chunkSize;
      const end = Math.min(start + chunkSize, file.size);
      fileReader.readAsArrayBuffer(file.slice(start, end));
    }

    loadNext();
  });
}
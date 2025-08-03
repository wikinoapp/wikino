// CSRFトークンを取得
export function csrfToken(): string {
  const meta = document.querySelector('meta[name="csrf-token"]');
  if (!meta) {
    throw new Error("CSRF token not found");
  }
  return meta.getAttribute("content") || "";
}
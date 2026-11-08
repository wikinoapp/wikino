import { commonmarkLanguage } from "@codemirror/lang-markdown";
import type * as MarkdownModule from "@codemirror/lang-markdown";
import type { MarkdownParser } from "@lezer/markdown";
import { describe, expect, it, vi } from "vitest";

import { markdownLanguage } from "./markdown-language";

// 非公開の解析器の一覧を隠し、通常の解析と拡張の組み立ては実際の解析器に任せる。
vi.mock("@codemirror/lang-markdown", async (importOriginal) => {
  const original = await importOriginal<typeof MarkdownModule>();
  const parser = new Proxy(original.commonmarkLanguage.parser as MarkdownParser, {
    get(target, key, receiver) {
      if (key === "inlineNames") return undefined;
      const value: unknown = Reflect.get(target, key, receiver);
      return typeof value === "function" ? value.bind(target) : value;
    },
  });
  return { ...original, commonmarkLanguage: { ...original.commonmarkLanguage, parser } };
});

describe("標準のリンク解析器を取得できないとき", () => {
  it.each([
    ["リンク", "[a](u)"],
    ["画像", "![画像](u)"],
    ["参照リンク", "[a][r]\n\n[r]: u"],
  ])("%sの構文木を標準の解析と同じに保つ", (_, body) => {
    expect(markdownLanguage.parser.parse(body).toString()).toBe(commonmarkLanguage.parser.parse(body).toString());
  });
});

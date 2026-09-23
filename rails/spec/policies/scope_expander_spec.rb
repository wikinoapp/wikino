# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe ScopeExpander do
  describe ".expand" do
    it "含意のないスコープはそのまま返すこと" do
      scopes = [Scope::TOPIC_READ, Scope::PAGE_READ]
      result = ScopeExpander.expand(scopes)

      expect(result).to contain_exactly(
        Scope::TOPIC_READ,
        Scope::PAGE_READ
      )
    end

    it "write スコープから read スコープに含意展開されること" do
      result = ScopeExpander.expand([Scope::TOPIC_WRITE])

      expect(result).to contain_exactly(
        Scope::TOPIC_WRITE,
        Scope::TOPIC_READ
      )
    end

    it "すべての write -> read 含意が展開されること" do
      scopes = [
        Scope::TOPIC_WRITE,
        Scope::TOPIC_MEMBER_WRITE,
        Scope::PAGE_WRITE,
        Scope::PAGE_TRASH_WRITE,
        Scope::DRAFT_PAGE_WRITE,
        Scope::SUGGESTION_WRITE,
        Scope::SUGGESTION_COMMENT_WRITE,
        Scope::SPACE_WRITE,
        Scope::SPACE_MEMBER_WRITE,
        Scope::ATTACHMENT_WRITE
      ]
      result = ScopeExpander.expand(scopes)

      expect(result).to include(
        Scope::TOPIC_READ,
        Scope::TOPIC_MEMBER_READ,
        Scope::PAGE_READ,
        Scope::PAGE_TRASH_READ,
        Scope::DRAFT_PAGE_READ,
        Scope::SUGGESTION_READ,
        Scope::SUGGESTION_COMMENT_READ,
        Scope::SPACE_READ,
        Scope::SPACE_MEMBER_READ,
        Scope::ATTACHMENT_READ
      )
    end

    it "space:admin が全リソーススコープに展開されること" do
      result = ScopeExpander.expand([Scope::SPACE_ADMIN])

      expect(result).to include(Scope::SPACE_ADMIN)
      ScopeExpander::ALL_RESOURCE_SCOPES.each do |scope|
        expect(result).to include(scope)
      end
    end

    it "重複するスコープが除去されること" do
      scopes = [Scope::TOPIC_READ, Scope::TOPIC_WRITE]
      result = ScopeExpander.expand(scopes)

      expect(result.count { |s| s == Scope::TOPIC_READ }).to eq(1)
    end

    it "空の配列を渡すと空の配列が返ること" do
      result = ScopeExpander.expand([])

      expect(result).to eq([])
    end

    it "削除・反映・クローズのスコープは他の権限を含意しないこと" do
      scopes = [
        Scope::TOPIC_DELETE,
        Scope::PAGE_TRASH_DELETE,
        Scope::DRAFT_PAGE_DELETE,
        Scope::SUGGESTION_APPLICATION_WRITE,
        Scope::SUGGESTION_CLOSURE_WRITE
      ]
      result = ScopeExpander.expand(scopes)

      expect(result).to match_array(scopes)
    end

    [
      ["page:trash", "page_trash:write", ["page_trash:write", "page_trash:read"]],
      ["page:restore", "page_trash:delete", ["page_trash:delete"]],
      ["suggestion:apply", "suggestion_application:write", ["suggestion_application:write"]],
      ["suggestion:close", "suggestion_closure:write", ["suggestion_closure:write"]]
    ].each do |legacy, canonical, expected|
      it "#{legacy}と#{canonical}が同じ正式スコープへ展開されること" do
        expect(ScopeExpander.expand([legacy])).to match_array(expected)
        expect(ScopeExpander.expand([canonical])).to match_array(expected)
      end

      it "#{legacy}と#{canonical}が混在しても重複せず、入力を変更しないこと" do
        scopes = [legacy, canonical, legacy].freeze

        expect(ScopeExpander.expand(scopes)).to match_array(expected)
        expect(scopes).to eq([legacy, canonical, legacy])
      end
    end

    it "space:adminは正式な新名を包括し、旧名を含まないこと" do
      result = ScopeExpander.expand([Scope::SPACE_ADMIN])

      expect(result).to include(
        "page_trash:read", "page_trash:write", "page_trash:delete",
        "suggestion_application:write", "suggestion_closure:write"
      )
      expect(result).not_to include("page:trash", "page:restore", "suggestion:apply", "suggestion:close")
    end

    it "page:writeはゴミ箱の権限を含意しないこと" do
      expect(ScopeExpander.expand([Scope::PAGE_WRITE])).to contain_exactly(
        Scope::PAGE_WRITE, Scope::PAGE_READ
      )
    end

    it "suggestion:writeは反映・クローズの権限を含意しないこと" do
      expect(ScopeExpander.expand([Scope::SUGGESTION_WRITE])).to contain_exactly(
        Scope::SUGGESTION_WRITE, Scope::SUGGESTION_READ
      )
    end

    it "未知のスコープや旧名の類似文字列から権限を展開しないこと" do
      scopes = ["unknown:write", "page:trash:write", "page:restore_extra", "suggestion:apply_extra", "suggestion:close_extra"]

      expect(ScopeExpander.expand(scopes)).to match_array(scopes)
    end

    it "draft_page:write は draft_page:delete を含意しないこと" do
      result = ScopeExpander.expand([Scope::DRAFT_PAGE_WRITE])

      expect(result).to contain_exactly(
        Scope::DRAFT_PAGE_WRITE,
        Scope::DRAFT_PAGE_READ
      )
    end
  end

  describe "正式スコープの一覧" do
    it "通常のactionはread・write・deleteに限定され、space:adminだけが例外であること" do
      canonical_scopes = ScopeExpander::ALL_RESOURCE_SCOPES + [Scope::SPACE_ADMIN]

      expect(ScopeExpander::ALL_RESOURCE_SCOPES).not_to include(Scope::SPACE_ADMIN)
      expect(Scope::SPACE_ADMIN).to eq("space:admin")
      canonical_scopes.each do |scope|
        next if scope == Scope::SPACE_ADMIN

        expect(scope).to match(/\A[a-z_]+:(read|write|delete)\z/)
      end
    end
  end
end

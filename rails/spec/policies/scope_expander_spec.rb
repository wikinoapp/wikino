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

    it "含意のないアクション（delete, trash, restore, apply, close）は展開されないこと" do
      scopes = [
        Scope::TOPIC_DELETE,
        Scope::PAGE_TRASH,
        Scope::PAGE_RESTORE,
        Scope::SUGGESTION_APPLY,
        Scope::SUGGESTION_CLOSE
      ]
      result = ScopeExpander.expand(scopes)

      expect(result).to match_array(scopes)
    end
  end
end

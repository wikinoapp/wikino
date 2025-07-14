# typed: strict
# frozen_string_literal: true

module Pages
  class SearchForm < ApplicationForm
    attribute :q, :string

    validates :q, length: {
      minimum: 2,
      maximum: 100,
      allow_blank: true,
      too_short: :too_short,
      too_long: :too_long
    }
    validates :q, format: {with: /\A[^<>]*\z/, message: :invalid}

    sig { returns(T::Boolean) }
    def query_present?
      !!q&.present?
    end

    sig { returns(T::Boolean) }
    def searchable?
      return false if invalid?
      return false unless query_present?

      # 検索オプションのみの場合は検索実行しない
      keyword_without_space_filters.present?
    end

    # space:指定子を解析してスペース識別子のリストを取得
    sig { returns(T::Array[String]) }
    def space_identifiers
      return [] if q.blank?

      q.not_nil!.scan(/space:(\S+)/).flatten
    end

    # space:指定子を除いたキーワードを取得
    sig { returns(String) }
    def keyword_without_space_filters
      return "" if q.blank?

      q.not_nil!.gsub(/space:\S+/, "").strip.squeeze(" ")
    end

    # スペースフィルターが指定されているかチェック
    sig { returns(T::Boolean) }
    def has_space_filters?
      space_identifiers.any?
    end
  end
end

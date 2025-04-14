# typed: strict
# frozen_string_literal: true

class Space < ApplicationModel
  IDENTIFIER_FORMAT = /\A[A-Za-z0-9-]+\z/
  # 識別子の最大文字数 (値に強い理由は無い)
  IDENTIFIER_MAX_LENGTH = 20
  # 識別子の予約語
  IDENTIFIER_RESERVED_WORDS = %w[www].freeze
  # 名前の最大文字数 (値に強い理由は無い)
  NAME_MAX_LENGTH = 30
end

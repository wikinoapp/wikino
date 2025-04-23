# typed: strict
# frozen_string_literal: true

class Topic < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  # 名前の最大文字数 (値に強い理由は無い)
  NAME_MAX_LENGTH = 30
  # 説明の最大文字数 (値に強い理由は無い)
  DESCRIPTION_MAX_LENGTH = 150

  const :database_id, T::Wikino::DatabaseId
  const :space, Space
end

# typed: strict
# frozen_string_literal: true

class User < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  ATNAME_FORMAT = /\A[A-Za-z0-9_]+\z/
  # アットネームの最大文字数 (値に強い理由は無い)
  ATNAME_MAX_LENGTH = 20
  # 名前の最大文字数 (値に強い理由は無い)
  NAME_MAX_LENGTH = 30
  # 説明の最大文字数 (値に強い理由は無い)
  DESCRIPTION_MAX_LENGTH = 150

  const :database_id, Types::DatabaseId
  const :email, String
  const :atname, String
  const :name, String
  const :description, String
  const :locale, Locale
  const :time_zone, String
end

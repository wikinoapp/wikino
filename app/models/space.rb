# typed: strict
# frozen_string_literal: true

class Space < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  IDENTIFIER_FORMAT = /\A[A-Za-z0-9-]+\z/
  # 識別子の最大文字数 (値に強い理由は無い)
  IDENTIFIER_MAX_LENGTH = 20
  # 識別子の予約語
  IDENTIFIER_RESERVED_WORDS = %w[www].freeze
  # 名前の最大文字数 (値に強い理由は無い)
  NAME_MAX_LENGTH = 30

  const :database_id, Types::DatabaseId
  const :identifier, String
  const :name, String
  const :plan, Plan
  const :joined_at, ActiveSupport::TimeWithZone
  const :can_create_topic, T.nilable(T::Boolean)

  sig { returns(T::Boolean) }
  def can_create_topic?
    can_create_topic.not_nil!
  end
end

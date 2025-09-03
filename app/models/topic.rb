# typed: strict
# frozen_string_literal: true

class Topic < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  # 名前の最大文字数 (値に強い理由は無い)
  NAME_MAX_LENGTH = 30
  # 説明の最大文字数 (値に強い理由は無い)
  DESCRIPTION_MAX_LENGTH = 150

  const :database_id, Types::DatabaseId
  const :number, Integer
  const :name, String
  const :description, String
  const :visibility, TopicVisibility
  const :can_update, T.nilable(T::Boolean)
  const :can_create_page, T.nilable(T::Boolean)
  const :space, Space

  sig { returns(T::Boolean) }
  def visibility_public?
    visibility == TopicVisibility::Public
  end

  sig { returns(T::Boolean) }
  def can_update?
    can_update.not_nil!
  end

  sig { returns(T::Boolean) }
  def can_create_page?
    can_create_page.not_nil!
  end
end

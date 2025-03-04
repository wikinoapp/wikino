# typed: strict
# frozen_string_literal: true

class PageEntity < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :database_id, T::Wikino::DatabaseId
  const :number, Integer
  const :title, T.nilable(String)
  const :body, String
  const :body_html, String
  const :modified_at, ActiveSupport::TimeWithZone
  const :published_at, T.nilable(ActiveSupport::TimeWithZone)
  const :pinned_at, T.nilable(ActiveSupport::TimeWithZone)
  const :space_entity, SpaceEntity
  const :topic_entity, TopicEntity
  const :viewer_can_update, T::Boolean

  alias_method :viewer_can_update?, :viewer_can_update

  sig { returns(T::Boolean) }
  def published?
    published_at.present?
  end

  sig { returns(T::Boolean) }
  def pinned?
    pinned_at.present?
  end

  sig { returns(T::Boolean) }
  def modified_after_published?
    published? && modified_at > published_at
  end
end

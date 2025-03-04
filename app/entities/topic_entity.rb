# typed: strict
# frozen_string_literal: true

class TopicEntity < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :database_id, T::Wikino::DatabaseId
  const :number, Integer
  const :name, String
  const :description, String
  const :visibility, TopicVisibility
  const :space_entity, SpaceEntity
  const :viewer_can_create_page, T::Boolean

  alias_method :viewer_can_create_page?, :viewer_can_create_page

  sig { returns(T::Boolean) }
  def visibility_public?
    visibility == TopicVisibility::Public
  end
end

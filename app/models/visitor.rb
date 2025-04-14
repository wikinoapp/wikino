# typed: strict
# frozen_string_literal: true

# ログインしていない人を表すモデル
class Visitor < ApplicationModel
  include ModelConcerns::Viewable

  sig { override.returns(String) }
  attr_accessor :serialized_locale

  sig { override.returns(String) }
  attr_accessor :time_zone

  sig { params(attributes: T::Hash[Symbol, T.untyped]).void }
  def initialize(attributes = {})
    attributes[:time_zone] = attributes[:time_zone].presence || "Asia/Tokyo"
    attributes[:serialized_locale] = attributes[:serialized_locale].presence || "ja"
    super
  end

  sig { override.returns(T::Boolean) }
  def signed_in?
    false
  end

  sig { override.returns(T.nilable(UserEntity)) }
  def user_entity
    nil
  end

  sig { override.params(space: Space).returns(ModelConcerns::SpaceViewable) }
  def space_viewer!(space:)
    SpaceVisitor.new(space:)
  end

  sig { override.params(space: Space).returns(T::Boolean) }
  def joined_space?(space:)
    false
  end

  sig { override.params(topic: Topic).returns(T::Boolean) }
  def can_view_topic?(topic:)
    topic.visibility_public?
  end

  sig { override.params(page: Page).returns(T::Boolean) }
  def can_trash_page?(page:)
    false
  end

  sig { override.returns(Topic::PrivateRelation) }
  def viewable_topics
    Topic.visibility_public
  end
end

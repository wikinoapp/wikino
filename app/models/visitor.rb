# typed: strict
# frozen_string_literal: true

# ログインしていない人を表すモデル
class Visitor
  extend T::Sig

  include ModelConcerns::Viewable

  sig { params(time_zone: String, locale: String).void }
  def initialize(time_zone: "Asia/Tokyo", locale: "ja")
    @time_zone = time_zone
    @locale = locale
  end

  sig { override.returns(String) }
  attr_reader :time_zone

  sig { returns(String) }
  attr_reader :locale

  sig { override.returns(ViewerLocale) }
  def viewer_locale
    ViewerLocale.deserialize(locale)
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

  sig { override.params(topic: TopicRecord).returns(T::Boolean) }
  def can_view_topic?(topic:)
    topic.visibility_public?
  end

  sig { override.params(page: Page).returns(T::Boolean) }
  def can_trash_page?(page:)
    false
  end

  sig { override.returns(Topic::PrivateRelation) }
  def viewable_topics
    TopicRecord.visibility_public
  end
end

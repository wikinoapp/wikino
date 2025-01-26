# typed: strict
# frozen_string_literal: true

class Visitor
  extend T::Sig

  include ModelConcerns::Viewable

  def initialize(time_zone: "Asia/Tokyo", locale: ViewerLocale::Ja)
    @time_zone = time_zone
    @locale = locale
  end

  sig { override.returns(String) }
  attr_reader :time_zone

  sig { override.returns(ViewerLocale) }
  attr_reader :locale

  sig { override.returns(T::Boolean) }
  def signed_in?
    false
  end

  sig { override.params(page: Page).returns(T::Boolean) }
  def can_view_page?(page:)
    page.topic.not_nil!.visibility_public?
  end

  sig { override.params(space: Space).returns(T::Boolean) }
  def can_view_trash?(space:)
    false
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

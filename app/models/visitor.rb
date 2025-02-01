# typed: strict
# frozen_string_literal: true

# ログインしていない人を表すモデル
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

  sig { override.params(space: Space).returns(T::Boolean) }
  def joined_space?(space:)
    false
  end

  sig { override.params(page: Page).returns(T::Boolean) }
  def can_view_page?(page:)
    page.topic.not_nil!.visibility_public?
  end

  sig { override.params(topic: Topic).returns(T::Boolean) }
  def can_view_topic?(topic:)
    topic.visibility_public?
  end

  sig { override.params(space: Space).returns(T::Boolean) }
  def can_view_trash?(space:)
    false
  end

  sig { override.params(topic: Topic).returns(T::Boolean) }
  def can_create_topic?(topic:)
    false
  end

  sig { override.params(topic: Topic).returns(T::Boolean) }
  def can_create_page?(topic:)
    false
  end

  sig { override.params(space: Space).returns(T::Boolean) }
  def can_create_bulk_restored_pages?(space:)
    false
  end

  sig { override.params(page: Page).returns(T::Boolean) }
  def can_update_draft_page?(page:)
    false
  end

  sig { override.params(page: Page).returns(T::Boolean) }
  def can_update_page?(page:)
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

  sig { override.returns(T.any(DraftPage::PrivateAssociationRelation, DraftPage::PrivateRelation)) }
  def active_draft_pages
    DraftPage.none
  end

  sig { override.params(space: Space, number: T.untyped).returns(Topic) }
  def find_topic_by_number!(space:, number:)
    Topic.visibility_public.find_by!(space:, number:)
  end
end

# typed: strict
# frozen_string_literal: true

# スペースに参加していない人を表すモデル
class SpaceVisitor
  extend T::Sig

  include ModelConcerns::SpaceViewable

  sig { params(space: Space).void }
  def initialize(space:)
    @space = space
  end

  sig { returns(Space) }
  attr_reader :space

  sig { override.returns(T.any(DraftPage::PrivateAssociationRelation, DraftPage::PrivateRelation)) }
  def draft_pages
    DraftPage.none
  end

  sig { override.returns(Page::PrivateAssociationRelation) }
  def showable_pages
    space.pages.active.joins(:topic).merge(Topic.visibility_public)
  end

  sig { override.returns(T.any(Topic::PrivateAssociationRelation, Topic::PrivateRelation)) }
  def joined_topics
    Topic.none
  end

  sig { override.returns(Topic::PrivateAssociationRelation) }
  def showable_topics
    space.topics.kept.visibility_public
  end

  sig { override.params(space: Space).returns(T::Boolean) }
  def can_update_space?(space:)
    false
  end

  sig { override.params(space: Space).returns(T::Boolean) }
  def can_export_space?(space:)
    false
  end

  sig { override.params(topic: Topic).returns(T::Boolean) }
  def can_update_topic?(topic:)
    false
  end

  sig { override.params(topic: T.nilable(Topic)).returns(T::Boolean) }
  def can_create_page?(topic:)
    false
  end

  sig { override.params(space: Space).returns(T::Boolean) }
  def can_create_bulk_restored_pages?(space:)
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
  def can_update_page?(page:)
    false
  end

  sig { override.returns(T::Boolean) }
  def can_create_topic?
    false
  end

  sig { override.params(page: Page).returns(T::Boolean) }
  def can_update_draft_page?(page:)
    false
  end
end

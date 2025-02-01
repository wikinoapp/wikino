# typed: strict
# frozen_string_literal: true

# ログインしていない人を表すモデル
class SpaceVisitor
  extend T::Sig

  include ModelConcerns::SpaceViewable

  sig { params(space: Space).void }
  def initialize(space:)
    @space = space
  end

  sig { returns(Space) }
  attr_reader :space

  sig { override.returns(Page::PrivateRelation) }
  def viewable_pages
    space.pages.active.joins(:topic).merge(Topic.visibility_public)
  end

  sig { override.returns(Topic::PrivateRelation) }
  def topics
    space.topics.visibility_public
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

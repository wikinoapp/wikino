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

  sig { override.returns(Page::PrivateAssociationRelation) }
  def viewable_pages
    space.pages.active.joins(:topic).merge(Topic.visibility_public)
  end

  sig { override.returns(Topic::PrivateAssociationRelation) }
  def topics
    space.topics.visibility_public
  end

  sig { override.params(number: T.untyped).returns(Topic) }
  def find_topic_by_number!(number:)
    topics.find_by!(number:)
  end

  sig { override.params(topic: T.nilable(Topic)).returns(T::Boolean) }
  def can_create_page?(topic:)
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

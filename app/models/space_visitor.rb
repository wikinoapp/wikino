# typed: strict
# frozen_string_literal: true

# スペースに参加していない人を表すモデル
class SpaceVisitor
  extend T::Sig

  include ModelConcerns::SpaceViewable

  sig { params(space: SpaceRecord).void }
  def initialize(space:)
    @space = space
  end

  sig { returns(SpaceRecord) }
  attr_reader :space

  sig { override.returns(T.any(DraftPageRecord::PrivateAssociationRelation, DraftPageRecord::PrivateRelation)) }
  def draft_page_records
    DraftPageRecord.none
  end

  sig { override.returns(T.any(TopicRecord::PrivateAssociationRelation, TopicRecord::PrivateRelation)) }
  def joined_topics
    TopicRecord.none
  end

  sig { override.params(space: SpaceRecord).returns(T::Boolean) }
  def can_update_space?(space:)
    false
  end

  sig { override.params(space: SpaceRecord).returns(T::Boolean) }
  def can_export_space?(space:)
    false
  end

  sig { override.params(topic: T.nilable(TopicRecord)).returns(T::Boolean) }
  def can_create_page?(topic:)
    false
  end

  sig { override.params(space: SpaceRecord).returns(T::Boolean) }
  def can_create_bulk_restored_pages?(space:)
    false
  end

  sig { override.params(page: PageRecord).returns(T::Boolean) }
  def can_view_page?(page:)
    page.topic_record.not_nil!.visibility_public?
  end

  sig { override.params(page: PageRecord).returns(T::Boolean) }
  def can_update_page?(page:)
    false
  end

  sig { override.params(page: PageRecord).returns(T::Boolean) }
  def can_update_draft_page?(page:)
    false
  end
end

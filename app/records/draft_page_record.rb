# typed: strict
# frozen_string_literal: true

class DraftPageRecord < ApplicationRecord
  include RecordConcerns::Pageable

  self.table_name = "draft_pages"

  belongs_to :space_record, foreign_key: :space_id
  belongs_to :topic_record, foreign_key: :topic_id
  belongs_to :page_record, foreign_key: :page_id
  belongs_to :space_member_record, foreign_key: :space_member_id

  sig { params(space_viewer: ModelConcerns::SpaceViewable).returns(DraftPageEntity) }
  def to_entity(space_viewer:)
    DraftPageEntity.new(
      database_id: id,
      modified_at:,
      page_entity: page_record.not_nil!.to_entity(space_viewer:)
    )
  end
end

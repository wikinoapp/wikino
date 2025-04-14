# typed: strict
# frozen_string_literal: true

class DraftPageRecord < ApplicationRecord
  include ModelConcerns::Pageable

  self.table_name = "draft_pages"

  belongs_to :space
  belongs_to :topic
  belongs_to :page
  belongs_to :space_member

  sig { params(space_viewer: ModelConcerns::SpaceViewable).returns(DraftPageEntity) }
  def to_entity(space_viewer:)
    DraftPageEntity.new(
      database_id: id,
      modified_at:,
      page_entity: page.not_nil!.to_entity(space_viewer:)
    )
  end
end
